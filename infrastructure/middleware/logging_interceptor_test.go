package middleware_test

import (
	"context"
	"net"
	"testing"

	userpb "go-feature-based-boilerplate/gen/pb/user"
	"go-feature-based-boilerplate/infrastructure/middleware"
	"go-feature-based-boilerplate/pkg/contextutil"
	"go-feature-based-boilerplate/pkg/logger"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestLoggingInterceptor_Success(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	req := &userpb.GetUserRequest{Id: 10}
	resp, err := interceptor(context.Background(), req, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)

	entries := logs.All()
	assert.Len(t, entries, 1)
	entry := entries[0]
	assert.Equal(t, zapcore.InfoLevel, entry.Level)
	assert.Equal(t, "gRPC request completed", entry.Message)

	fieldMap := entry.ContextMap()
	assert.Equal(t, "grpc", fieldMap["rpc.system"])
	assert.Equal(t, "user.v1.UserService", fieldMap["rpc.service"])
	assert.Equal(t, "GetUser", fieldMap["rpc.method"])
	assert.Equal(t, int64(0), fieldMap["rpc.grpc.status_code"])
	assert.Equal(t, "OK", fieldMap["status"])
}

func TestLoggingInterceptor_ClientError(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/CreateUser"}

	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "invalid email address")
	})

	assert.Error(t, err)
	entries := logs.All()
	assert.Len(t, entries, 1)
	entry := entries[0]
	assert.Equal(t, zapcore.WarnLevel, entry.Level)
	assert.Equal(t, "gRPC client error", entry.Message)

	fieldMap := entry.ContextMap()
	assert.Equal(t, "InvalidArgument", fieldMap["status"])
	assert.Equal(t, "invalid email address", fieldMap["error.message"])
}

func TestLoggingInterceptor_ServerError(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.Internal, "database connection failed")
	})

	assert.Error(t, err)
	entries := logs.All()
	assert.Len(t, entries, 1)
	entry := entries[0]
	assert.Equal(t, zapcore.ErrorLevel, entry.Level)
	assert.Equal(t, "gRPC server error", entry.Message)

	fieldMap := entry.ContextMap()
	assert.Equal(t, "Internal", fieldMap["status"])
	assert.Equal(t, "database connection failed", fieldMap["error.message"])
}

func TestLoggingInterceptor_PayloadMasking(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/CreateUser"}

	req := &userpb.CreateUserRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "supersecretpassword123",
	}

	_, _ = interceptor(context.Background(), req, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	entries := logs.All()
	assert.Len(t, entries, 1)
	fieldMap := entries[0].ContextMap()

	payload, ok := fieldMap["request"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Alice", payload["name"])
	assert.Equal(t, "alice@example.com", payload["email"])
	assert.Equal(t, "[REDACTED]", payload["password"])

	// Response must NOT be logged on successful request
	assert.Nil(t, fieldMap["response"])
}

func TestLoggingInterceptor_ResponseLoggedOnError(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/CreateUser"}

	respPayload := map[string]any{
		"token":   "secret-bearer-token",
		"message": "partially processed with error",
	}

	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return respPayload, status.Error(codes.Internal, "internal server failure")
	})

	assert.Error(t, err)
	entries := logs.All()
	assert.Len(t, entries, 1)
	fieldMap := entries[0].ContextMap()

	// Response MUST be logged on error with sensitive keys masked
	respField, ok := fieldMap["response"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "[REDACTED]", respField["token"])
	assert.Equal(t, "partially processed with error", respField["message"])
}

func TestLoggingInterceptor_ContextEnrichment(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(zapLogger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	md := metadata.Pairs("x-request-id", "req-corr-777", "user-agent", "grpc-client/v2.1")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedReqID string
	var capturedRPCSystem string
	var capturedRPCService string
	var capturedRPCMethod string
	var capturedUserAgent string

	_, _ = interceptor(ctx, nil, info, func(innerCtx context.Context, req any) (any, error) {
		if id, ok := contextutil.GetRequestID(innerCtx); ok {
			capturedReqID = id
		}
		if rpcMeta, ok := logger.GetRPCMetadata(innerCtx); ok {
			capturedRPCSystem = rpcMeta.System
			capturedRPCService = rpcMeta.Service
			capturedRPCMethod = rpcMeta.Method
			capturedUserAgent = rpcMeta.UserAgent
		}
		return "ok", nil
	})

	assert.Equal(t, "req-corr-777", capturedReqID)
	assert.Equal(t, "grpc", capturedRPCSystem)
	assert.Equal(t, "user.v1.UserService", capturedRPCService)
	assert.Equal(t, "GetUser", capturedRPCMethod)
	assert.Equal(t, "grpc-client/v2.1", capturedUserAgent)

	entries := logs.All()
	assert.Len(t, entries, 1)
	fieldMap := entries[0].ContextMap()
	assert.Equal(t, "req-corr-777", fieldMap["request_id"])
	assert.Equal(t, "grpc-client/v2.1", fieldMap["user_agent"])
	assert.NotNil(t, fieldMap["duration_ms"])
}

func TestLoggingInterceptor_PeerIP_And_RequestID(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	interceptor := middleware.LoggingInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	// Attach peer IP
	dummyAddr, _ := net.ResolveTCPAddr("tcp", "192.168.1.100:54321")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: dummyAddr})

	// Attach request ID in metadata
	md := metadata.Pairs("x-request-id", "req-xyz-999")
	ctx = metadata.NewIncomingContext(ctx, md)

	_, _ = interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	entries := logs.All()
	assert.Len(t, entries, 1)
	fieldMap := entries[0].ContextMap()

	assert.Equal(t, "192.168.1.100:54321", fieldMap["client_ip"])
	assert.Equal(t, "req-xyz-999", fieldMap["request_id"])
}
