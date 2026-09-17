package logger

import (
	"context"
	"os"

	"go-feature-based-boilerplate/pkg/contextutil"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// zapLogger is the Zap implementation of the Logger interface.
type zapLogger struct {
	base    *zap.Logger
	service string
	env     string
	host    string
}

// NewZapLogger creates a new Logger instance wrapping Uber Zap.
// It wraps the provided *zap.Logger with caller skip = 1 so that caller file and line
// numbers accurately reflect the call site outside of this wrapper.
func NewZapLogger(base *zap.Logger, service string, env string) Logger {
	hostname, _ := os.Hostname()
	return &zapLogger{
		base:    base.WithOptions(zap.AddCallerSkip(1)),
		service: service,
		env:     env,
		host:    hostname,
	}
}

// Debug logs a debug-level message formatted as JSON with all extracted context fields.
func (l *zapLogger) Debug(ctx context.Context, msg string) {
	fields := l.extractContextFields(ctx)
	l.base.Debug(msg, fields...)
}

// Info logs an info-level message formatted as JSON with all extracted context fields.
func (l *zapLogger) Info(ctx context.Context, msg string) {
	fields := l.extractContextFields(ctx)
	l.base.Info(msg, fields...)
}

// Warn logs a warning-level message formatted as JSON with all extracted context fields.
func (l *zapLogger) Warn(ctx context.Context, msg string) {
	fields := l.extractContextFields(ctx)
	l.base.Warn(msg, fields...)
}

// Error logs an error-level message formatted as JSON with all extracted context fields.
// If an error is provided, it is attached as the "error" field.
func (l *zapLogger) Error(ctx context.Context, msg string, err ...error) {
	fields := l.extractContextFields(ctx)
	if len(err) > 0 && err[0] != nil {
		fields = append(fields, zap.Error(err[0]))
	}
	l.base.Error(msg, fields...)
}

// Fatal logs a fatal-level message and terminates the process via os.Exit(1).
func (l *zapLogger) Fatal(ctx context.Context, msg string, err ...error) {
	fields := l.extractContextFields(ctx)
	if len(err) > 0 && err[0] != nil {
		fields = append(fields, zap.Error(err[0]))
	}
	l.base.Fatal(msg, fields...)
}

// With returns a new Logger that adds a key-value pair to every log entry emitted.
func (l *zapLogger) With(key string, value any) Logger {
	return &zapLogger{
		base:    l.base.With(zap.Any(key, value)),
		service: l.service,
		env:     l.env,
		host:    l.host,
	}
}

// WithData returns a new Logger that attaches the given structured object under the "data" key.
func (l *zapLogger) WithData(data any) Logger {
	return &zapLogger{
		base:    l.base.With(zap.Any("data", data)),
		service: l.service,
		env:     l.env,
		host:    l.host,
	}
}

// extractContextFields automatically extracts standardized microservice fields from context.
func (l *zapLogger) extractContextFields(ctx context.Context) []zap.Field {
	// Allocate reasonable capacity for standard microservice fields
	fields := make([]zap.Field, 0, 12)

	// 1. Service metadata
	if l.service != "" {
		fields = append(fields, zap.String("service", l.service))
	}
	if l.env != "" {
		fields = append(fields, zap.String("env", l.env))
	}
	if l.host != "" {
		fields = append(fields, zap.String("host", l.host))
	}

	if ctx == nil {
		return fields
	}

	// 2. Distributed Tracing (OpenTelemetry W3C trace_id, span_id)
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.IsValid() {
		fields = append(fields,
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// 3. Request ID (from contextutil or gRPC incoming metadata)
	var requestID string
	if id, ok := contextutil.GetRequestID(ctx); ok && id != "" {
		requestID = id
	} else if md, ok := metadata.FromIncomingContext(ctx); ok {
		if reqIDs := md.Get("x-request-id"); len(reqIDs) > 0 && reqIDs[0] != "" {
			requestID = reqIDs[0]
		}
	}
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	// 4. Authenticated Identity (user_id, role)
	if userID, ok := contextutil.GetUserID(ctx); ok {
		fields = append(fields, zap.Uint("user_id", userID))
	}
	if role, ok := contextutil.GetRole(ctx); ok && role != "" {
		fields = append(fields, zap.String("role", role))
	}

	// 5. RPC Transport Metadata (rpc.system, rpc.service, rpc.method, client_ip, user_agent)
	hasClientIP := false
	if rpcMeta, ok := GetRPCMetadata(ctx); ok {
		if rpcMeta.System != "" {
			fields = append(fields, zap.String("rpc.system", rpcMeta.System))
		}
		if rpcMeta.Service != "" {
			fields = append(fields, zap.String("rpc.service", rpcMeta.Service))
		}
		if rpcMeta.Method != "" {
			fields = append(fields, zap.String("rpc.method", rpcMeta.Method))
		}
		if rpcMeta.ClientIP != "" {
			fields = append(fields, zap.String("client_ip", rpcMeta.ClientIP))
			hasClientIP = true
		}
		if rpcMeta.UserAgent != "" {
			fields = append(fields, zap.String("user_agent", rpcMeta.UserAgent))
		}
	}

	// 6. Peer / Client IP fallback
	if !hasClientIP {
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			fields = append(fields, zap.String("client_ip", p.Addr.String()))
		}
	}

	return fields
}
