package logger

import (
	"context"
	"fmt"
	"os"

	"go-feature-based-boilerplate/pkg/contextutil"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// New constructs a unified production-ready microservice Logger from Config.
// All log entries are formatted strictly as JSON with ISO8601 timestamps and emitted to stdout.
func New(cfg Config) (Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stack_trace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var zapCfg zap.Config
	if cfg.Environment == "production" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.Development = false
	}

	zapCfg.Encoding = "json"
	zapCfg.EncoderConfig = encoderConfig
	zapCfg.Level = zap.NewAtomicLevelAt(level)
	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	base, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, fmt.Errorf("logger: build zap: %w", err)
	}

	return NewWithZap(base, cfg.ServiceName, cfg.Environment), nil
}

// NewWithZap wraps an existing *zap.Logger into the standard Logger interface.
// Useful for unit testing with zaptest observer cores or custom Zap sinks.
func NewWithZap(base *zap.Logger, service string, env string) Logger {
	hostname, _ := os.Hostname()

	// Pre-attach static process metadata so it is not recomputed or allocated per log call
	var staticFields []zap.Field
	if service != "" {
		staticFields = append(staticFields, zap.String("service", service))
	}
	if env != "" {
		staticFields = append(staticFields, zap.String("env", env))
	}
	if hostname != "" {
		staticFields = append(staticFields, zap.String("host", hostname))
	}

	prepared := base.WithOptions(zap.AddCallerSkip(1))
	if len(staticFields) > 0 {
		prepared = prepared.With(staticFields...)
	}

	return &zapLogger{
		base:    prepared,
		service: service,
		env:     env,
		host:    hostname,
	}
}

// NewZapLogger is an alias for NewWithZap for backward compatibility.
func NewZapLogger(base *zap.Logger, service string, env string) Logger {
	return NewWithZap(base, service, env)
}

// NewNop returns a no-op Logger satisfying the Logger interface.
func NewNop() Logger {
	return NewWithZap(zap.NewNop(), "nop", "test")
}

// Debug logs a debug-level message formatted as JSON with all extracted context fields.
func (l *zapLogger) Debug(ctx context.Context, msg string) {
	l.logWithSkip(ctx, 0, zapcore.DebugLevel, msg)
}

// Info logs an info-level message formatted as JSON with all extracted context fields.
func (l *zapLogger) Info(ctx context.Context, msg string) {
	l.logWithSkip(ctx, 0, zapcore.InfoLevel, msg)
}

// Warn logs a warning-level message formatted as JSON with all extracted context fields.
func (l *zapLogger) Warn(ctx context.Context, msg string) {
	l.logWithSkip(ctx, 0, zapcore.WarnLevel, msg)
}

// Error logs an error-level message formatted as JSON with all extracted context fields.
// If an error is provided, it is attached as the "error" field.
func (l *zapLogger) Error(ctx context.Context, msg string, err ...error) {
	l.logWithSkip(ctx, 0, zapcore.ErrorLevel, msg, err...)
}

// Fatal logs a fatal-level message and terminates the process via os.Exit(1).
func (l *zapLogger) Fatal(ctx context.Context, msg string, err ...error) {
	l.logWithSkip(ctx, 0, zapcore.FatalLevel, msg, err...)
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

// WithFields returns a new Logger that attaches multiple key-value pairs to every log entry emitted.
func (l *zapLogger) WithFields(fields map[string]any) Logger {
	if len(fields) == 0 {
		return l
	}
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &zapLogger{
		base:    l.base.With(zapFields...),
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

// Named returns a new Logger with a sub-logger namespace.
func (l *zapLogger) Named(name string) Logger {
	return &zapLogger{
		base:    l.base.Named(name),
		service: l.service,
		env:     l.env,
		host:    l.host,
	}
}

// Sync flushes any buffered log entries.
func (l *zapLogger) Sync() error {
	return l.base.Sync()
}

// Desugar returns the underlying *zap.Logger for third-party integrations (e.g. gRPC interceptors, GORM).
func (l *zapLogger) Desugar() *zap.Logger {
	return l.base
}

// logCaller implements internalCallerLogger for caller depth correction in package-level functions.
func (l *zapLogger) logCaller(ctx context.Context, extraSkip int, level string, msg string, err ...error) {
	var zapLvl zapcore.Level
	switch level {
	case "debug":
		zapLvl = zapcore.DebugLevel
	case "info":
		zapLvl = zapcore.InfoLevel
	case "warn":
		zapLvl = zapcore.WarnLevel
	case "error":
		zapLvl = zapcore.ErrorLevel
	case "fatal":
		zapLvl = zapcore.FatalLevel
	default:
		zapLvl = zapcore.InfoLevel
	}
	l.logWithSkip(ctx, extraSkip, zapLvl, msg, err...)
}

// logWithSkip executes the core logging operation with level check fast-path and caller skip correction.
func (l *zapLogger) logWithSkip(ctx context.Context, extraSkip int, level zapcore.Level, msg string, err ...error) {
	// Zero-allocation fast-path: if the level is not enabled, return immediately
	if !l.base.Core().Enabled(level) {
		return
	}

	fields := l.extractContextFields(ctx)

	// Attach errors if provided
	if len(err) > 0 {
		var validErrs []error
		for _, e := range err {
			if e != nil {
				validErrs = append(validErrs, e)
			}
		}
		if len(validErrs) == 1 {
			fields = append(fields, zap.Error(validErrs[0]))
		} else if len(validErrs) > 1 {
			fields = append(fields, zap.Error(validErrs[0]))
			fields = append(fields, zap.Errors("extra_errors", validErrs[1:]))
		}
	}

	logger := l.base
	if extraSkip > 0 {
		logger = logger.WithOptions(zap.AddCallerSkip(extraSkip))
	}

	switch level {
	case zapcore.DebugLevel:
		logger.Debug(msg, fields...)
	case zapcore.InfoLevel:
		logger.Info(msg, fields...)
	case zapcore.WarnLevel:
		logger.Warn(msg, fields...)
	case zapcore.ErrorLevel:
		logger.Error(msg, fields...)
	case zapcore.FatalLevel:
		logger.Fatal(msg, fields...)
	}
}

// extractContextFields automatically extracts standardized microservice correlation fields from context.
func (l *zapLogger) extractContextFields(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	// Allocate capacity for standard dynamic microservice correlation fields
	fields := make([]zap.Field, 0, 8)

	// 1. Distributed Tracing (OpenTelemetry W3C trace_id, span_id)
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.IsValid() {
		fields = append(fields,
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// 2. Request ID (from contextutil or gRPC incoming metadata)
	var requestID string
	if id, ok := contextutil.GetRequestID(ctx); ok && id != "" {
		requestID = id
	} else if md, ok := metadata.FromIncomingContext(ctx); ok {
		if reqIDs := md.Get("x-request-id"); len(reqIDs) > 0 && reqIDs[0] != "" {
			requestID = reqIDs[0]
		} else if corrIDs := md.Get("x-correlation-id"); len(corrIDs) > 0 && corrIDs[0] != "" {
			requestID = corrIDs[0]
		}
	}
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	// 3. Authenticated Identity (user_id, role)
	if userID, ok := contextutil.GetUserID(ctx); ok {
		fields = append(fields, zap.Uint("user_id", userID))
	}
	if role, ok := contextutil.GetRole(ctx); ok && role != "" {
		fields = append(fields, zap.String("role", role))
	}

	// 4. RPC Transport Metadata (rpc.system, rpc.service, rpc.method, client_ip, user_agent)
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

	// 5. Peer / Client IP fallback
	if !hasClientIP {
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			fields = append(fields, zap.String("client_ip", p.Addr.String()))
		}
	}

	return fields
}

func parseLevel(s string) (zapcore.Level, error) {
	switch s {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("logger: unknown level %q", s)
	}
}
