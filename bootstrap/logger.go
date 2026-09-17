package bootstrap

import (
	infraConfig "go-feature-based-boilerplate/infrastructure/config"
	pkgLogger "go-feature-based-boilerplate/pkg/logger"

	"go.uber.org/zap"
)

// ProvideLogger constructs a *zap.Logger from the application config.
func ProvideLogger(cfg *infraConfig.Config) (*zap.Logger, error) {
	return pkgLogger.NewZap(pkgLogger.Config{
		Level:       cfg.Logger.Level,
		Environment: cfg.App.Environment,
	})
}
