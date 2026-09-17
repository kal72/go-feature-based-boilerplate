package bootstrap

import (
	"time"

	authpb "go-feature-based-boilerplate/gen/pb/auth"
	healthpb "go-feature-based-boilerplate/gen/pb/health"
	userpb "go-feature-based-boilerplate/gen/pb/user"
	"go-feature-based-boilerplate/infrastructure/config"
	"go-feature-based-boilerplate/infrastructure/middleware"
	authHandler "go-feature-based-boilerplate/internal/auth/handler"
	healthHandler "go-feature-based-boilerplate/internal/health/handler"
	userHandler "go-feature-based-boilerplate/internal/user/handler"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// NewGRPCServer assembles the gRPC server with all interceptors and service registrations.
func NewGRPCServer(
	cfg *config.Config,
	logger *zap.Logger,
	userH *userHandler.Handler,
	authH *authHandler.Handler,
	healthH *healthHandler.Handler,
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

	// Register business services.
	userpb.RegisterUserServiceServer(server, userH)
	authpb.RegisterAuthServiceServer(server, authH)
	healthpb.RegisterHealthServiceServer(server, healthH)

	// Register reflection service for client discovery in non-production environments.
	if cfg.App.Environment != "production" {
		reflection.Register(server)
	}

	return server
}
