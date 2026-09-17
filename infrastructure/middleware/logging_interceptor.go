package middleware

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go-feature-based-boilerplate/pkg/contextutil"
	"go-feature-based-boilerplate/pkg/logger"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// LoggingInterceptor is a gRPC unary server interceptor that logs each request
// in structured JSON format with OpenTelemetry semantic conventions, execution latency,
// status codes, peer IP, request ID, sanitized request payload, and response payload (on error).
// It also enriches the context with RPCMetadata so downstream handlers and usecases
// automatically inherit transport context.
func LoggingInterceptor(zapLogger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		serviceName, methodName := parseFullMethod(info.FullMethod)

		// 1. Extract Client IP
		var clientIP string
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			clientIP = p.Addr.String()
		}

		// 2. Extract Request ID and User Agent from metadata
		var requestID string
		var userAgent string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if reqIDs := md.Get("x-request-id"); len(reqIDs) > 0 {
				requestID = reqIDs[0]
			}
			if uas := md.Get("user-agent"); len(uas) > 0 {
				userAgent = uas[0]
			} else if uas := md.Get("grpcgateway-user-agent"); len(uas) > 0 {
				userAgent = uas[0]
			}
		}

		// 3. Enrich Context with RPCMetadata & RequestID for downstream loggers/usecases
		rpcMeta := logger.RPCMetadata{
			System:    "grpc",
			Service:   serviceName,
			Method:    methodName,
			ClientIP:  clientIP,
			UserAgent: userAgent,
		}
		ctx = logger.WithRPCMetadata(ctx, rpcMeta)
		if requestID != "" {
			ctx = contextutil.WithRequestID(ctx, requestID)
		}

		// Execute downstream handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		durationMs := float64(duration.Microseconds()) / 1000.0

		st, _ := status.FromError(err)
		code := st.Code()

		fields := []zap.Field{
			zap.String("rpc.system", "grpc"),
			zap.String("rpc.service", serviceName),
			zap.String("rpc.method", methodName),
			zap.Int("rpc.grpc.status_code", int(code)),
			zap.String("status", code.String()),
			zap.Float64("duration_ms", durationMs),
		}

		if clientIP != "" {
			fields = append(fields, zap.String("client_ip", clientIP))
		}
		if userAgent != "" {
			fields = append(fields, zap.String("user_agent", userAgent))
		}
		if requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		// 4. OpenTelemetry Trace & Span ID
		spanCtx := trace.SpanFromContext(ctx).SpanContext()
		if spanCtx.IsValid() {
			fields = append(fields,
				zap.String("trace_id", spanCtx.TraceID().String()),
				zap.String("span_id", spanCtx.SpanID().String()),
			)
		}

		// 5. Authenticated Identity (if set during interceptor chain)
		if userID, ok := contextutil.GetUserID(ctx); ok {
			fields = append(fields, zap.Uint("user_id", userID))
		}
		if role, ok := contextutil.GetRole(ctx); ok && role != "" {
			fields = append(fields, zap.String("role", role))
		}

		// 6. Request Payload with sensitive data masked (always logged)
		if sanitizedReq := sanitizePayload(req); sanitizedReq != nil {
			fields = append(fields,
				zap.Any("request", sanitizedReq),
				zap.Any("rpc.request.payload", sanitizedReq), // backward compatibility
			)
		}

		// 7. Response Payload with sensitive data masked (logged ONLY on error)
		if code != codes.OK {
			if sanitizedResp := sanitizePayload(resp); sanitizedResp != nil {
				fields = append(fields, zap.Any("response", sanitizedResp))
			}
		}

		// 8. Error Details if request failed
		if err != nil {
			fields = append(fields, zap.String("error.message", st.Message()))
		}

		// 9. Dynamic Log Level based on gRPC status code
		switch code {
		case codes.OK:
			zapLogger.Info("gRPC request completed", fields...)
		case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
			codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
			zapLogger.Warn("gRPC client error", fields...)
		default:
			zapLogger.Error("gRPC server error", fields...)
		}

		return resp, err
	}
}

// parseFullMethod decomposes "/user.v1.UserService/GetUser" into ("user.v1.UserService", "GetUser")
func parseFullMethod(fullMethod string) (string, string) {
	trimmed := strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return fullMethod, fullMethod
}

var sensitiveKeys = map[string]bool{
	"password":      true,
	"old_password":  true,
	"new_password":  true,
	"token":         true,
	"refresh_token": true,
	"access_token":  true,
	"secret":        true,
	"secret_key":    true,
	"authorization": true,
	"credit_card":   true,
	"cvv":           true,
	"card_number":   true,
}

func isSensitiveKey(key string) bool {
	if sensitiveKeys[key] {
		return true
	}
	for s := range sensitiveKeys {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

// sanitizePayload converts the request to a JSON-compatible map and redacts sensitive keys.
func sanitizePayload(req any) any {
	if req == nil {
		return nil
	}

	var rawBytes []byte
	var err error

	if pm, ok := req.(proto.Message); ok {
		rawBytes, err = protojson.Marshal(pm)
	} else {
		rawBytes, err = json.Marshal(req)
	}
	if err != nil {
		return "[unserializable payload]"
	}

	var parsed any
	if err := json.Unmarshal(rawBytes, &parsed); err != nil {
		return "[invalid json payload]"
	}

	return maskSensitive(parsed)
}

func maskSensitive(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, item := range val {
			if isSensitiveKey(strings.ToLower(k)) {
				result[k] = "[REDACTED]"
			} else {
				result[k] = maskSensitive(item)
			}
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = maskSensitive(item)
		}
		return result
	default:
		return v
	}
}
