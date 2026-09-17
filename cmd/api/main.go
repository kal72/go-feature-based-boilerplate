package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-feature-based-boilerplate/bootstrap"
)

func main() {
	ctx := context.Background()

	app, cleanup, err := bootstrap.InitializeApp(ctx)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer cleanup()

	// Start gRPC and HTTP gateway concurrently.
	go app.StartGRPC()
	go app.StartHTTPGateway()

	// Block until OS interrupt or termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Give in-flight requests up to 30 seconds to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	app.Shutdown(shutdownCtx)
}
