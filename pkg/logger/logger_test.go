package logger_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"

	"go-feature-based-boilerplate/pkg/contextutil"
	"go-feature-based-boilerplate/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestZapLogger_FormatAndContextFields(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	baseZap := zap.New(core)

	log := logger.NewZapLogger(baseZap, "order-service", "production")

	// 1. Setup rich context
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	ctx = contextutil.WithRequestID(ctx, "req-12345")
	ctx = contextutil.WithUserID(ctx, 42)
	ctx = contextutil.WithRole(ctx, "admin")

	rpcMeta := logger.RPCMetadata{
		System:    "grpc",
		Service:   "order.v1.OrderService",
		Method:    "CreateOrder",
		ClientIP:  "192.168.1.50",
		UserAgent: "grpc-go/1.62.0",
	}
	ctx = logger.WithRPCMetadata(ctx, rpcMeta)

	// 2. Call Info
	log.Info(ctx, "order created successfully")

	entries := recorded.All()
	require.Len(t, entries, 1)
	entry := entries[0]

	assert.Equal(t, zapcore.InfoLevel, entry.Level)
	assert.Equal(t, "order created successfully", entry.Message)

	fields := entry.ContextMap()
	assert.Equal(t, "order-service", fields["service"])
	assert.Equal(t, "production", fields["env"])
	assert.NotEmpty(t, fields["host"])
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", fields["trace_id"])
	assert.Equal(t, "00f067aa0ba902b7", fields["span_id"])
	assert.Equal(t, "req-12345", fields["request_id"])
	assert.Equal(t, uint64(42), fields["user_id"])
	assert.Equal(t, "admin", fields["role"])
	assert.Equal(t, "grpc", fields["rpc.system"])
	assert.Equal(t, "order.v1.OrderService", fields["rpc.service"])
	assert.Equal(t, "CreateOrder", fields["rpc.method"])
	assert.Equal(t, "192.168.1.50", fields["client_ip"])
	assert.Equal(t, "grpc-go/1.62.0", fields["user_agent"])
}

func TestZapLogger_LevelsAndError(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	baseZap := zap.New(core)
	log := logger.NewZapLogger(baseZap, "test-svc", "test")

	ctx := context.Background()

	log.Debug(ctx, "debug msg")
	log.Warn(ctx, "warn msg")

	testErr := errors.New("database timeout")
	log.Error(ctx, "operation failed", testErr)

	// Multi-error check
	secondErr := errors.New("connection reset")
	log.Error(ctx, "multiple failures", testErr, secondErr)

	entries := recorded.All()
	require.Len(t, entries, 4)

	assert.Equal(t, zapcore.DebugLevel, entries[0].Level)
	assert.Equal(t, "debug msg", entries[0].Message)

	assert.Equal(t, zapcore.WarnLevel, entries[1].Level)
	assert.Equal(t, "warn msg", entries[1].Message)

	assert.Equal(t, zapcore.ErrorLevel, entries[2].Level)
	assert.Equal(t, "operation failed", entries[2].Message)
	fields := entries[2].ContextMap()
	assert.Equal(t, "database timeout", fields["error"])

	// Validate multiple errors
	multiFields := entries[3].ContextMap()
	assert.Equal(t, "database timeout", multiFields["error"])
	assert.NotNil(t, multiFields["extra_errors"])
}

func TestZapLogger_FastPathLevelCheck(t *testing.T) {
	// Logger configured at WarnLevel
	core, recorded := observer.New(zapcore.WarnLevel)
	baseZap := zap.New(core)
	log := logger.NewZapLogger(baseZap, "test-svc", "production")

	ctx := context.Background()

	// Debug and Info should be bypassed via fast-path check
	log.Debug(ctx, "debug ignored")
	log.Info(ctx, "info ignored")

	assert.Equal(t, 0, recorded.Len())

	// Warn should be recorded
	log.Warn(ctx, "warning triggered")
	assert.Equal(t, 1, recorded.Len())
	assert.Equal(t, "warning triggered", recorded.All()[0].Message)
}

func TestZapLogger_WithAndWithDataAndWithFields(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	baseZap := zap.New(core)
	log := logger.NewZapLogger(baseZap, "test-svc", "test")

	type OrderPayload struct {
		ID     string  `json:"id"`
		Amount float64 `json:"amount"`
	}

	payload := OrderPayload{ID: "ord-99", Amount: 150000}

	subLog := log.
		With("custom_key", "custom_val").
		WithFields(map[string]any{
			"batch_id": "b-100",
			"priority": "high",
		}).
		WithData(payload)

	subLog.Info(context.Background(), "processing transaction")

	entries := recorded.All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	assert.Equal(t, "custom_val", fields["custom_key"])
	assert.Equal(t, "b-100", fields["batch_id"])
	assert.Equal(t, "high", fields["priority"])

	// Validate data payload
	dataMap, ok := fields["data"].(map[string]any)
	if !ok {
		dataJSON, err := json.Marshal(fields["data"])
		require.NoError(t, err)
		var unmarshaled map[string]any
		require.NoError(t, json.Unmarshal(dataJSON, &unmarshaled))
		dataMap = unmarshaled
	}
	assert.Equal(t, "ord-99", dataMap["id"])
	assert.Equal(t, float64(150000), dataMap["amount"])
}

func TestZapLogger_NamedAndSyncAndDesugar(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	baseZap := zap.New(core)
	log := logger.NewZapLogger(baseZap, "test-svc", "test")

	namedLog := log.Named("subsystem")
	namedLog.Info(context.Background(), "named logger message")

	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "subsystem", entries[0].LoggerName)

	// Sync should not panic or fail
	require.NoError(t, log.Sync())

	// Desugar returns underlying *zap.Logger
	if zl, ok := log.(interface{ Desugar() *zap.Logger }); ok {
		assert.NotNil(t, zl.Desugar())
	}
}

func TestZapLogger_MetadataAndPeerFallback(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	baseZap := zap.New(core)
	log := logger.NewZapLogger(baseZap, "test-svc", "test")

	// Set peer IP in context without RPCMetadata
	dummyAddr, _ := net.ResolveTCPAddr("tcp", "10.0.0.1:8080")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: dummyAddr})

	// Set incoming gRPC metadata for request ID
	md := metadata.Pairs("x-request-id", "grpc-md-req-456")
	ctx = metadata.NewIncomingContext(ctx, md)

	log.Info(ctx, "peer and md test")

	entries := recorded.All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	assert.Equal(t, "10.0.0.1:8080", fields["client_ip"])
	assert.Equal(t, "grpc-md-req-456", fields["request_id"])
}

func TestLogger_New(t *testing.T) {
	// Development config
	devLog, err := logger.New(logger.Config{
		Level:       "debug",
		Environment: "development",
		ServiceName: "dev-service",
	})
	require.NoError(t, err)
	assert.NotNil(t, devLog)

	// Production config
	prodLog, err := logger.New(logger.Config{
		Level:       "info",
		Environment: "production",
		ServiceName: "prod-service",
	})
	require.NoError(t, err)
	assert.NotNil(t, prodLog)

	// Nop logger implements Logger
	nop := logger.NewNop()
	assert.NotNil(t, nop)
	require.NoError(t, nop.Sync())

	// Unknown level falls back to info with error
	_, err = logger.New(logger.Config{
		Level: "unknown_level",
	})
	assert.Error(t, err)
}

func TestPackageLevelLogging_AndDefaultLogger(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	baseZap := zap.New(core)
	appLogger := logger.NewZapLogger(baseZap, "pkg-service", "test")

	// Set package-level default
	logger.SetDefault(appLogger)
	assert.Equal(t, appLogger, logger.L())

	ctx := context.Background()

	logger.Debug(ctx, "pkg debug")
	logger.Info(ctx, "pkg info")
	logger.Warn(ctx, "pkg warn")
	logger.Error(ctx, "pkg error", errors.New("err"))

	logger.With("key", "val").Info(ctx, "fluent pkg info")
	logger.WithFields(map[string]any{"f1": 1}).Info(ctx, "fluent pkg fields")
	logger.WithData("data_val").Info(ctx, "fluent pkg data")
	logger.Named("pkg-sub").Info(ctx, "fluent pkg named")
	require.NoError(t, logger.Sync())

	entries := recorded.All()
	assert.Len(t, entries, 8)
	assert.Equal(t, "pkg debug", entries[0].Message)
	assert.Equal(t, "pkg info", entries[1].Message)
	assert.Equal(t, "pkg warn", entries[2].Message)
	assert.Equal(t, "pkg error", entries[3].Message)
}

func TestContextLogger_WithAndFromContext(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	baseZap := zap.New(core)
	defaultLog := logger.NewZapLogger(baseZap, "default-svc", "test")
	logger.SetDefault(defaultLog)

	// Fallback to default when not set in context
	emptyCtx := context.Background()
	assert.NotNil(t, logger.FromContext(emptyCtx))

	// Scoped logger attached to context
	scopedLog := defaultLog.With("tenant", "tenant-abc")
	ctx := logger.WithContext(emptyCtx, scopedLog)

	logger.FromContext(ctx).Info(ctx, "scoped tenant message")

	entries := recorded.All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	assert.Equal(t, "tenant-abc", fields["tenant"])
}
