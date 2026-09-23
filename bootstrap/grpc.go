package bootstrap

import (
	"time"

	"go-feature-based-boilerplate/infrastructure/config"
	"go-feature-based-boilerplate/infrastructure/middleware"
	pkgLogger "go-feature-based-boilerplate/pkg/logger"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// NewGRPCServer assembles the gRPC server with all interceptors and registers services via Services.
func NewGRPCServer(
	cfg *config.Config,
	logger pkgLogger.Logger,
	services *Services,
) *grpc.Server {
	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Timeout:           20 * time.Second,
		}),
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor(logger),
			middleware.LoggingInterceptor(logger),
			middleware.TimeoutInterceptor(cfg.Server.DefaultTimeout),
			middleware.AuthInterceptor(cfg),
			middleware.ErrorInterceptor(),
		),
	)

	// Register all business services via unified Services collection.
	services.RegisterGRPC(server)

	// Register reflection service for client discovery in non-production environments.
	if cfg.App.Environment != "production" {
		reflection.Register(server)
	}

	return server
}
