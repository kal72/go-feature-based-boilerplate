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

	entries := recorded.All()
	require.Len(t, entries, 3)

	assert.Equal(t, zapcore.DebugLevel, entries[0].Level)
	assert.Equal(t, "debug msg", entries[0].Message)

	assert.Equal(t, zapcore.WarnLevel, entries[1].Level)
	assert.Equal(t, "warn msg", entries[1].Message)

	assert.Equal(t, zapcore.ErrorLevel, entries[2].Level)
	assert.Equal(t, "operation failed", entries[2].Message)
	fields := entries[2].ContextMap()
	assert.Equal(t, "database timeout", fields["error"])
}

func TestZapLogger_WithAndWithData(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	baseZap := zap.New(core)
	log := logger.NewZapLogger(baseZap, "test-svc", "test")

	type OrderPayload struct {
		ID     string  `json:"id"`
		Amount float64 `json:"amount"`
	}

	payload := OrderPayload{ID: "ord-99", Amount: 150000}

	subLog := log.With("custom_key", "custom_val").WithData(payload)
	subLog.Info(context.Background(), "processing transaction")

	entries := recorded.All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	assert.Equal(t, "custom_val", fields["custom_key"])

	// Validate data payload
	dataMap, ok := fields["data"].(map[string]any)
	if !ok {
		// Could be serialized struct
		dataJSON, err := json.Marshal(fields["data"])
		require.NoError(t, err)
		var unmarshaled map[string]any
		require.NoError(t, json.Unmarshal(dataJSON, &unmarshaled))
		dataMap = unmarshaled
	}
	assert.Equal(t, "ord-99", dataMap["id"])
	assert.Equal(t, float64(150000), dataMap["amount"])
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

func TestFactory_NewZapAndNew(t *testing.T) {
	// Development config
	devZap, err := logger.NewZap(logger.Config{
		Level:       "debug",
		Environment: "development",
	})
	require.NoError(t, err)
	assert.NotNil(t, devZap)

	// Production config
	prodZap, err := logger.NewZap(logger.Config{
		Level:       "info",
		Environment: "production",
	})
	require.NoError(t, err)
	assert.NotNil(t, prodZap)

	// New complete logger
	appLogger, err := logger.New(logger.Config{
		Level:       "warn",
		Environment: "development",
	}, "test-service")
	require.NoError(t, err)
	assert.NotNil(t, appLogger)

	// Nop logger
	nop := logger.NewNop()
	assert.NotNil(t, nop)

	// Unknown level falls back to info with error
	_, err = logger.NewZap(logger.Config{
		Level: "unknown_level",
	})
	assert.Error(t, err)
}
