package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger initialisation options for building a zap.Logger.
type Config struct {
	Level       string // debug | info | warn | error
	Environment string // development | production
}

// NewZap creates a *zap.Logger configured to output strictly structured JSON.
// Regardless of the environment, log output is always formatted as JSON with
// ISO8601 UTC timestamps to ensure predictable parsing by log aggregators.
func NewZap(cfg Config) (*zap.Logger, error) {
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

	logger, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, fmt.Errorf("logger: build: %w", err)
	}

	return logger, nil
}

// New creates a complete standardized Logger from Config and service name.
func New(cfg Config, service string) (Logger, error) {
	zapLog, err := NewZap(cfg)
	if err != nil {
		return nil, err
	}
	return NewZapLogger(zapLog, service, cfg.Environment), nil
}

// NewNop returns a no-op *zap.Logger, useful in tests.
func NewNop() *zap.Logger {
	return zap.NewNop()
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
