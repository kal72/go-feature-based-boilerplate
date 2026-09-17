//go:build wireinject

package bootstrap

import (
	"context"

	infraConfig "go-feature-based-boilerplate/infrastructure/config"
	"go-feature-based-boilerplate/infrastructure/database/postgres"
	"go-feature-based-boilerplate/infrastructure/telemetry"
	"go-feature-based-boilerplate/internal/auth"
	"go-feature-based-boilerplate/internal/health"
	"go-feature-based-boilerplate/internal/user"

	"github.com/google/wire"
)

// serverSet wires bootstrap servers and application lifecycle container.
var serverSet = wire.NewSet(
	NewGRPCServer,
	NewHTTPGateway,
	NewApp,
)

// InitializeApp is the Wire injection entry point.
// wire.Build declares the full dependency graph; wire generates wire_gen.go.
func InitializeApp(ctx context.Context) (*App, func(), error) {
	wire.Build(
		// Config (from environment variables)
		infraConfig.ProviderSet,

		// Logger
		ProvideLogger,

		// Infrastructure
		postgres.ProviderSet,
		telemetry.ProviderSet,

		// Features
		user.ProviderSet,
		auth.ProviderSet,
		health.ProviderSet,

		// Bootstrap
		serverSet,
	)
	return nil, nil, nil
}
