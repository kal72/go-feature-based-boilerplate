package logger

import (
	"context"
	"sync/atomic"
)

// Config holds logger initialisation options for building a Logger.
type Config struct {
	Level       string // debug | info | warn | error
	Environment string // development | production
	ServiceName string // logical name of the microservice
}

// Logger defines an enterprise-grade structured JSON logger tailored for microservices.
// To enforce strict observability standards across the distributed architecture,
// log methods do not accept ad-hoc field variadics (e.g., fields ...Field).
// Instead, correlation identifiers (trace_id, span_id, request_id, user_id, rpc metadata)
// are automatically extracted from context.Context.
// Optional business payloads can be appended using the fluent With(), WithFields(), or WithData() methods.
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

	// WithFields creates a child Logger enriched with multiple key-value pairs.
	WithFields(fields map[string]any) Logger

	// WithData creates a child Logger enriched with structured data stored under the "data" key.
	WithData(data any) Logger

	// Named creates a child Logger with a sub-logger namespace.
	Named(name string) Logger

	// Sync flushes any buffered log entries to underlying storage/streams.
	Sync() error
}

var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(NewNop())
}

// SetDefault replaces the package-level default logger.
func SetDefault(l Logger) {
	if l != nil {
		defaultLogger.Store(l)
	}
}

// L returns the package-level default logger.
func L() Logger {
	if l, ok := defaultLogger.Load().(Logger); ok && l != nil {
		return l
	}
	return NewNop()
}

// Debug writes a debug-level log message using the default logger.
func Debug(ctx context.Context, msg string) {
	if zl, ok := L().(internalCallerLogger); ok {
		zl.logCaller(ctx, 1, "debug", msg)
		return
	}
	L().Debug(ctx, msg)
}

// Info writes an info-level log message using the default logger.
func Info(ctx context.Context, msg string) {
	if zl, ok := L().(internalCallerLogger); ok {
		zl.logCaller(ctx, 1, "info", msg)
		return
	}
	L().Info(ctx, msg)
}

// Warn writes a warning-level log message using the default logger.
func Warn(ctx context.Context, msg string) {
	if zl, ok := L().(internalCallerLogger); ok {
		zl.logCaller(ctx, 1, "warn", msg)
		return
	}
	L().Warn(ctx, msg)
}

// Error writes an error-level log message using the default logger.
func Error(ctx context.Context, msg string, err ...error) {
	if zl, ok := L().(internalCallerLogger); ok {
		zl.logCaller(ctx, 1, "error", msg, err...)
		return
	}
	L().Error(ctx, msg, err...)
}

// Fatal writes a fatal-level log message using the default logger and terminates via os.Exit(1).
func Fatal(ctx context.Context, msg string, err ...error) {
	if zl, ok := L().(internalCallerLogger); ok {
		zl.logCaller(ctx, 1, "fatal", msg, err...)
		return
	}
	L().Fatal(ctx, msg, err...)
}

// With creates a child Logger enriched with an additional key-value pair from default logger.
func With(key string, value any) Logger {
	return L().With(key, value)
}

// WithFields creates a child Logger enriched with multiple key-value pairs from default logger.
func WithFields(fields map[string]any) Logger {
	return L().WithFields(fields)
}

// WithData creates a child Logger enriched with structured data stored under the "data" key.
func WithData(data any) Logger {
	return L().WithData(data)
}

// Named creates a child Logger with a sub-logger namespace using the default logger.
func Named(name string) Logger {
	return L().Named(name)
}

// Sync flushes any buffered log entries of the default logger.
func Sync() error {
	return L().Sync()
}

type loggerCtxKey struct{}

// WithContext returns a new context with the provided Logger attached.
func WithContext(ctx context.Context, l Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

// FromContext extracts the Logger from the context. If no Logger is present,
// it returns the package-level default logger (L()).
func FromContext(ctx context.Context) Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerCtxKey{}).(Logger); ok && l != nil {
			return l
		}
	}
	return L()
}

// internalCallerLogger is an unexported interface implemented by concrete wrappers
// allowing caller depth correction when invoking package-level log helpers.
type internalCallerLogger interface {
	logCaller(ctx context.Context, extraSkip int, level string, msg string, err ...error)
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
