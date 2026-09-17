package logger

import (
	"context"
)

// Logger defines an enterprise-grade structured JSON logger tailored for microservices.
// To enforce strict observability standards across the distributed architecture,
// log methods do not accept ad-hoc field variadics (e.g., fields ...Field).
// Instead, correlation identifiers (trace_id, span_id, request_id, user_id, rpc metadata)
// are automatically extracted from the context.Context.
// Optional business payloads can be appended using the fluent With() or WithData() methods.
type Logger interface {
	// Debug writes a debug-level log message in JSON format, enriched with context metadata.
	Debug(ctx context.Context, msg string)

	// Info writes an info-level log message in JSON format, enriched with context metadata.
	Info(ctx context.Context, msg string)

	// Warn writes a warning-level log message in JSON format, enriched with context metadata.
	Warn(ctx context.Context, msg string)

	// Error writes an error-level log message in JSON format, enriched with context metadata
	// and error stack traces.
	Error(ctx context.Context, msg string, err ...error)

	// Fatal writes a fatal-level log message and terminates the process via os.Exit(1).
	Fatal(ctx context.Context, msg string, err ...error)

	// With creates a child Logger enriched with an additional key-value pair.
	With(key string, value any) Logger

	// WithData creates a child Logger enriched with structured data stored under the "data" key.
	WithData(data any) Logger
}

// RPCMetadata stores RPC transport-level metadata injected by network interceptors (e.g. gRPC).
// These attributes follow OpenTelemetry semantic conventions for RPC systems.
type RPCMetadata struct {
	// System identifies the RPC system, e.g. "grpc".
	System string
	// Service identifies the target RPC service name, e.g. "user.v1.UserService".
	Service string
	// Method identifies the target RPC procedure name, e.g. "GetUser".
	Method string
	// ClientIP identifies the caller's network IP address.
	ClientIP string
	// UserAgent identifies the caller's client binary / SDK version.
	UserAgent string
}

type rpcContextKey struct{}

// WithRPCMetadata injects RPC transport metadata into the context.
// This allows downstream handlers, usecases, and repositories to automatically inherit
// transport metadata without passing transport parameters manually.
func WithRPCMetadata(ctx context.Context, meta RPCMetadata) context.Context {
	return context.WithValue(ctx, rpcContextKey{}, meta)
}

// GetRPCMetadata extracts RPC transport metadata from the context if available.
func GetRPCMetadata(ctx context.Context) (RPCMetadata, bool) {
	meta, ok := ctx.Value(rpcContextKey{}).(RPCMetadata)
	return meta, ok
}
