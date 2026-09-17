package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"go-feature-based-boilerplate/infrastructure/config"
	"go-feature-based-boilerplate/pkg/routine"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// App is the application container. It holds all top-level server handles and
// is responsible for starting and stopping them in the correct order.
type App struct {
	cfg            *config.Config
	logger         *zap.Logger
	db             *gorm.DB
	grpcServer     *grpc.Server
	httpServer     *http.Server
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// NewApp constructs the application container.
func NewApp(
	cfg *config.Config,
	logger *zap.Logger,
	db *gorm.DB,
	grpcServer *grpc.Server,
	httpServer *http.Server,
	tracerProvider *sdktrace.TracerProvider,
	meterProvider *sdkmetric.MeterProvider,
) *App {
	// Register structured Zap logger for all unhandled goroutine panics.
	routine.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
		logger.Error("routine: recovered from unhandled panic",
			zap.Any("panic", r),
			zap.String("stack", string(stack)),
		)
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
	addr := fmt.Sprintf(":%d", a.cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		a.logger.Fatal("grpc: listen", zap.String("addr", addr), zap.Error(err))
	}
	a.logger.Info("grpc server started", zap.String("addr", addr))
	if err := a.grpcServer.Serve(lis); err != nil {
		a.logger.Error("grpc server stopped", zap.Error(err))
	}
}

// StartHTTPGateway starts the HTTP/REST gateway server in the current goroutine.
// Call this inside a goroutine from main.
func (a *App) StartHTTPGateway() {
	addr := fmt.Sprintf(":%d", a.cfg.Server.HTTPPort)
	a.logger.Info("http gateway started", zap.String("addr", addr))
	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error("http gateway stopped", zap.Error(err))
	}
}

// Shutdown gracefully stops all servers and releases resources.
// Should be called after receiving a termination signal.
func (a *App) Shutdown(ctx context.Context) {
	a.logger.Info("shutting down...")

	// 1. Drain HTTP gateway first so ingress traffic stops and in-flight gateway calls finish.
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Error("http server shutdown error", zap.Error(err))
	}

	// 2. Stop gRPC server gracefully.
	a.grpcServer.GracefulStop()

	// 3. Wait for all active background goroutines to complete before closing data connections.
	if err := routine.WaitForShutdown(ctx); err != nil {
		a.logger.Warn("routine: shutdown timed out while waiting for background tasks", zap.Error(err))
	}

	// 4. Close database connection pool.
	if sqlDB, err := a.db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	// 5. Flush telemetry pipelines.
	if err := a.tracerProvider.Shutdown(ctx); err != nil {
		a.logger.Error("tracer provider shutdown error", zap.Error(err))
	}
	if err := a.meterProvider.Shutdown(ctx); err != nil {
		a.logger.Error("meter provider shutdown error", zap.Error(err))
	}

	// 6. Flush buffered log entries.
	_ = a.logger.Sync()
}
