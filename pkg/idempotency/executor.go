package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Option configures Executor behavior.
type Option func(*executorConfig)

type executorConfig struct {
	lockTTL     time.Duration
	completeTTL time.Duration
}

// WithLockTTL overrides the in-progress lock lease duration.
// If an operation crashes or hangs, the lock will expire after this duration.
// Defaults to 2 minutes if not specified.
func WithLockTTL(ttl time.Duration) Option {
	return func(c *executorConfig) {
		if ttl > 0 {
			c.lockTTL = ttl
		}
	}
}

// WithCompleteTTL overrides the duration for which completed responses are cached.
// Defaults to 24 hours if not specified.
func WithCompleteTTL(ttl time.Duration) Option {
	return func(c *executorConfig) {
		if ttl > 0 {
			c.completeTTL = ttl
		}
	}
}

// Executor wraps business operations with idempotency guarantees using Go Generics.
// It ensures that identical requests with the same idempotency key are executed
// exactly once, returning cached responses for subsequent calls.
type Executor[T any] struct {
	storage Storage
	cfg     executorConfig
}

// NewExecutor creates a new type-safe idempotency Executor for type T.
//
// Example usage:
//
//	exec := idempotency.NewExecutor[*entity.Payment](redisStorage,
//	    idempotency.WithLockTTL(1*time.Minute),
//	    idempotency.WithCompleteTTL(48*time.Hour),
//	)
//	payment, err := exec.Execute(ctx, req.IdempotencyKey, func(execCtx context.Context) (*entity.Payment, error) {
//	    return paymentRepo.Create(execCtx, ...)
//	})
func NewExecutor[T any](storage Storage, opts ...Option) *Executor[T] {
	cfg := executorConfig{
		lockTTL:     2 * time.Minute,
		completeTTL: 24 * time.Hour,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Executor[T]{
		storage: storage,
		cfg:     cfg,
	}
}

// Execute wraps fn with idempotency protection based on the provided key.
//
// Behavior:
//  1. Pass-through: If key is empty (""), fn is executed directly without idempotency checks.
//  2. Cache Hit: If key exists with StatusCompleted, the cached response is deserialized
//     and returned immediately without executing fn.
//  3. Conflict Detection: If key exists with StatusInProgress, ErrConcurrentRequest is returned.
//  4. Execution & Cache: If key is new, an in-progress lock is acquired. On success, the response
//     is cached with StatusCompleted. On failure, the lock is removed so the client may retry.
func (e *Executor[T]) Execute(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// If no idempotency key was supplied by client, pass through directly
	if key == "" {
		return fn(ctx)
	}

	// 1. Check existing record
	record, err := e.storage.Get(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("idempotency: check record: %w", err)
	}

	if record != nil {
		switch record.Status {
		case StatusCompleted:
			var cachedVal T
			if len(record.Response) > 0 {
				if err := json.Unmarshal(record.Response, &cachedVal); err != nil {
					return zero, fmt.Errorf("idempotency: unmarshal cached response: %w", err)
				}
			}
			return cachedVal, nil

		case StatusInProgress:
			return zero, ErrConcurrentRequest

		default:
			// Fallthrough for unrecognized status
		}
	}

	// 2. Acquire lock for new execution
	acquired, err := e.storage.Lock(ctx, key, e.cfg.lockTTL)
	if err != nil {
		return zero, fmt.Errorf("idempotency: acquire lock: %w", err)
	}
	if !acquired {
		return zero, ErrConcurrentRequest
	}

	// 3. Execute business logic
	val, err := fn(ctx)
	if err != nil {
		// Release key so client can retry with corrected payload or after transient fix
		_ = e.storage.Delete(ctx, key)
		return zero, err
	}

	// 4. Serialize and persist completed response
	respBytes, err := json.Marshal(val)
	if err != nil {
		return zero, fmt.Errorf("idempotency: marshal response: %w", err)
	}

	completedRecord := &Record{
		Status:    StatusCompleted,
		Response:  respBytes,
		CreatedAt: time.Now(),
	}

	if err := e.storage.Set(ctx, key, completedRecord, e.cfg.completeTTL); err != nil {
		// Log or return error if saving completed cache failed
		return val, fmt.Errorf("idempotency: save completed record: %w", err)
	}

	return val, nil
}
