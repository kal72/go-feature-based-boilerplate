package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"go-feature-based-boilerplate/infrastructure/config"
	pkgLogger "go-feature-based-boilerplate/pkg/logger"
	"go-feature-based-boilerplate/pkg/routine"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// App is the application container. It holds all top-level server handles and
// is responsible for starting and stopping them in the correct order.
type App struct {
	cfg            *config.Config
	logger         pkgLogger.Logger
	db             *gorm.DB
	grpcServer     *grpc.Server
	httpServer     *http.Server
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// NewApp constructs the application container.
func NewApp(
	cfg *config.Config,
	logger pkgLogger.Logger,
	db *gorm.DB,
	grpcServer *grpc.Server,
	httpServer *http.Server,
	tracerProvider *sdktrace.TracerProvider,
	meterProvider *sdkmetric.MeterProvider,
) *App {
	// Register structured logger for all unhandled goroutine panics.
	routine.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
		logger.
			With("panic", r).
			With("stack", string(stack)).
			Error(ctx, "routine: recovered from unhandled panic")
	})

	return &App{
		cfg:            cfg,
		logger:         logger,
		db:             db,
		grpcServer:     grpcServer,
		httpServer:     httpServer,
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
	}
}

// StartGRPC starts the gRPC server in the current goroutine.
// Call this inside a goroutine from main.
func (a *App) StartGRPC() {
	ctx := context.Background()
	addr := fmt.Sprintf(":%d", a.cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		a.logger.With("addr", addr).Fatal(ctx, "grpc: listen", err)
	}
	a.logger.With("addr", addr).Info(ctx, "grpc server started")
	if err := a.grpcServer.Serve(lis); err != nil {
		a.logger.Error(ctx, "grpc server stopped", err)
	}
}

// StartHTTPGateway starts the HTTP/REST gateway server in the current goroutine.
// Call this inside a goroutine from main.
func (a *App) StartHTTPGateway() {
	ctx := context.Background()
	addr := fmt.Sprintf(":%d", a.cfg.Server.HTTPPort)
	a.logger.With("addr", addr).Info(ctx, "http gateway started")
	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error(ctx, "http gateway stopped", err)
	}
}

// Shutdown gracefully stops all servers and releases resources.
// Should be called after receiving a termination signal.
func (a *App) Shutdown(ctx context.Context) {
	a.logger.Info(ctx, "shutting down...")

	// 1. Drain HTTP gateway first so ingress traffic stops and in-flight gateway calls finish.
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Error(ctx, "http server shutdown error", err)
	}

	// 2. Stop gRPC server gracefully.
	a.grpcServer.GracefulStop()

	// 3. Wait for all active background goroutines to complete before closing data connections.
	if err := routine.WaitForShutdown(ctx); err != nil {
		a.logger.With("error", err.Error()).Warn(ctx, "routine: shutdown timed out while waiting for background tasks")
	}

	// 4. Close database connection pool.
	if sqlDB, err := a.db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	// 5. Flush telemetry pipelines.
	if err := a.tracerProvider.Shutdown(ctx); err != nil {
		a.logger.Error(ctx, "tracer provider shutdown error", err)
	}
	if err := a.meterProvider.Shutdown(ctx); err != nil {
		a.logger.Error(ctx, "meter provider shutdown error", err)
	}

	// 6. Flush buffered log entries.
	_ = a.logger.Sync()
}
