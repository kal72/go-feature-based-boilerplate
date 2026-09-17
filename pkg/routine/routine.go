// Package routine provides concurrency primitives and goroutine lifecycle helpers
// designed for production Go microservices. It safeguards against uncaught panics,
// preserves context metadata across detached background executions, and coordinates
// with the server's graceful shutdown process.
package routine

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// PanicHandler defines the callback signature invoked when a goroutine panics.
type PanicHandler func(ctx context.Context, r any, stack []byte)

var (
	// tracker synchronizes active background goroutines for graceful shutdown.
	tracker sync.WaitGroup

	// activeCount tracks the current number of running goroutines.
	activeCount int64

	// mu protects panicHandler registration.
	mu sync.RWMutex

	// panicHandler is the global handler called when a goroutine recovers from panic.
	panicHandler PanicHandler = defaultPanicHandler
)

// defaultPanicHandler logs panic details and stack trace to standard log output.
func defaultPanicHandler(ctx context.Context, r any, stack []byte) {
	log.Printf("[routine] recovered from panic: %v\nStack trace:\n%s", r, string(stack))
}

// SetPanicHandler registers a custom panic handler (e.g., to record via Zap logger).
func SetPanicHandler(handler PanicHandler) {
	mu.Lock()
	defer mu.Unlock()
	if handler != nil {
		panicHandler = handler
	}
}

// getPanicHandler safely retrieves the current panic handler.
func getPanicHandler() PanicHandler {
	mu.RLock()
	defer mu.RUnlock()
	return panicHandler
}

// Go launches a new goroutine managed by the routine lifecycle.
//
// Features:
//  1. Automatic Panic Recovery: If fn panics, it is safely recovered and logged
//     via the configured PanicHandler, preventing the entire process from crashing.
//  2. Shutdown Tracking: Increments the global tracker so that server shutdown
//     waits for this goroutine to complete cleanly.
//  3. Context Propagation: Passes the given ctx directly into fn.
//  4. Prometheus Metrics: Observes active goroutines, task count, and duration.
func Go(ctx context.Context, fn func(ctx context.Context)) {
	tracker.Add(1)
	atomic.AddInt64(&activeCount, 1)
	metricIncActive()

	go func() {
		start := time.Now()
		defer func() {
			duration := time.Since(start).Seconds()
			metricDecActive()
			metricIncTasks("go")
			metricObserveDuration("go", duration)
			atomic.AddInt64(&activeCount, -1)
			tracker.Done()
			if r := recover(); r != nil {
				metricIncPanics()
				stack := debug.Stack()
				getPanicHandler()(ctx, r, stack)
			}
		}()

		fn(ctx)
	}()
}

// GoDetached launches a background goroutine whose lifecycle is detached from
// the parent request context's cancellation, while preserving all context values.
//
// This is critical for fire-and-forget background jobs (e.g., audit logging,
// sending notifications, publishing metrics) that must continue executing even
// if the client disconnects or the original HTTP/gRPC request times out.
//
// Internally, it leverages context.WithoutCancel to decouple cancellation signals
// while preserving trace spans, correlation request IDs, user claims, and baggage.
func GoDetached(ctx context.Context, fn func(bgCtx context.Context)) {
	bgCtx := context.WithoutCancel(ctx)
	Go(bgCtx, fn)
}

// Safe wraps a parameterless function with panic recovery and telemetry tracking.
// It is useful when launching lightweight inline functions: `go routine.Safe(fn)`.
func Safe(fn func()) {
	tracker.Add(1)
	atomic.AddInt64(&activeCount, 1)
	metricIncActive()

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metricDecActive()
		metricIncTasks("safe")
		metricObserveDuration("safe", duration)
		atomic.AddInt64(&activeCount, -1)
		tracker.Done()
		if r := recover(); r != nil {
			metricIncPanics()
			stack := debug.Stack()
			getPanicHandler()(context.Background(), r, stack)
		}
	}()

	fn()
}

// ActiveCount returns the current number of active goroutines managed by routine.
func ActiveCount() int64 {
	return atomic.LoadInt64(&activeCount)
}

// WaitForShutdown waits for all active goroutines launched via Go, GoDetached,
// and Safe to complete, or until ctx is canceled / times out.
//
// This should be invoked during server graceful shutdown before closing
// database connection pools or telemetry exporters.
func WaitForShutdown(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		tracker.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		remaining := ActiveCount()
		return fmt.Errorf("routine: shutdown timed out with %d goroutines still active: %w", remaining, ctx.Err())
	}
}
