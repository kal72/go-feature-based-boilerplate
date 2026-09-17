package routine

import (
	"context"
	"math/rand/v2"
	"time"
)

// RetryConfig defines exponential backoff and retry behavior for operations.
type RetryConfig struct {
	// MaxAttempts is the maximum total number of execution attempts (initial attempt + retries).
	// Must be >= 1. Defaults to 3 if <= 0.
	MaxAttempts int

	// InitialInterval is the pause duration before the first retry attempt.
	// Defaults to 100ms if <= 0.
	InitialInterval time.Duration

	// BackoffFactor is the multiplier applied to the interval after each failed attempt.
	// Defaults to 2.0 if <= 1.0.
	BackoffFactor float64

	// MaxInterval caps the maximum backoff delay between retry attempts.
	// Defaults to 5s if <= 0.
	MaxInterval time.Duration

	// Jitter enables randomized perturbation (up to +25% delay) to avoid thundering herd spikes.
	Jitter bool

	// RetryIf is an optional filter function. If specified, only errors returning true
	// will trigger subsequent retries. Non-retryable errors immediately return without retrying.
	RetryIf func(err error) bool
}

// applyDefaults populates missing or invalid configuration values with production defaults.
func (c *RetryConfig) applyDefaults() {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.InitialInterval <= 0 {
		c.InitialInterval = 100 * time.Millisecond
	}
	if c.BackoffFactor <= 1.0 {
		c.BackoffFactor = 2.0
	}
	if c.MaxInterval <= 0 {
		c.MaxInterval = 5 * time.Second
	}
}

// Retry executes the given function synchronously, automatically retrying upon failure
// according to the provided RetryConfig.
//
// If the context is canceled during execution or between backoff delays, Retry aborts
// and returns ctx.Err(). If all retry attempts are exhausted, the last error encountered is returned.
//
// Example usage:
//
//	err := routine.Retry(ctx, routine.RetryConfig{
//	    MaxAttempts:     3,
//	    InitialInterval: 50 * time.Millisecond,
//	    Jitter:          true,
//	}, func(ctx context.Context) error {
//	    return externalClient.Call(ctx)
//	})
func Retry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error {
	cfg.applyDefaults()

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metricIncTasks("retry")
		metricObserveDuration("retry", duration)
	}()

	currentInterval := cfg.InitialInterval

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		// Reached maximum attempts or non-retryable error
		if attempt == cfg.MaxAttempts {
			return err
		}
		if cfg.RetryIf != nil && !cfg.RetryIf(err) {
			return err
		}

		// Compute delay with optional jitter
		sleepDuration := currentInterval
		if cfg.Jitter {
			jitterLimit := sleepDuration / 4
			if jitterLimit > 0 {
				sleepDuration += time.Duration(rand.Int64N(int64(jitterLimit)))
			}
		}

		select {
		case <-time.After(sleepDuration):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Calculate next exponential backoff interval capped at MaxInterval
		nextInterval := time.Duration(float64(currentInterval) * cfg.BackoffFactor)
		if nextInterval > cfg.MaxInterval {
			nextInterval = cfg.MaxInterval
		}
		currentInterval = nextInterval
	}

	return nil
}

// GoWithRetry launches a managed background goroutine that retries fn according to cfg.
// Execution is protected against uncaught panics and tracked for graceful shutdown.
//
// Example usage:
//
//	routine.GoWithRetry(ctx, routine.RetryConfig{MaxAttempts: 5}, func(ctx context.Context) error {
//	    return webhookClient.Dispatch(ctx, payload)
//	})
func GoWithRetry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) {
	Go(ctx, func(gCtx context.Context) {
		_ = Retry(gCtx, cfg, fn)
	})
}

// GoDetachedWithRetry launches a background goroutine detached from the parent request
// context's cancellation, retrying fn according to cfg until completed or max attempts reached.
//
// Context metadata (trace spans, correlation IDs, user tokens) is preserved via context.WithoutCancel.
//
// Example usage:
//
//	routine.GoDetachedWithRetry(reqCtx, routine.RetryConfig{MaxAttempts: 3}, func(bgCtx context.Context) error {
//	    return auditLogService.Record(bgCtx, event)
//	})
func GoDetachedWithRetry(ctx context.Context, cfg RetryConfig, fn func(bgCtx context.Context) error) {
	bgCtx := context.WithoutCancel(ctx)
	GoWithRetry(bgCtx, cfg, fn)
}
