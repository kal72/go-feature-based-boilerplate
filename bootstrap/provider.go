package bootstrap

import (
	infraConfig "go-feature-based-boilerplate/infrastructure/config"
	pkgLogger "go-feature-based-boilerplate/pkg/logger"
)

// ProvideLogger constructs the unified application microservice Logger from the application config.
func ProvideLogger(cfg *infraConfig.Config) (pkgLogger.Logger, error) {
	log, err := pkgLogger.New(pkgLogger.Config{
		Level:       cfg.Logger.Level,
		Environment: cfg.App.Environment,
		ServiceName: cfg.App.Name,
	})
	if err != nil {
		return nil, err
	}

	pkgLogger.SetDefault(log)
	return log, nil
}
