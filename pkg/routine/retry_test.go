package routine_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/routine"

	"github.com/stretchr/testify/assert"
)

func TestRetry_SuccessImmediate(t *testing.T) {
	var attempts int
	err := routine.Retry(context.Background(), routine.RetryConfig{
		MaxAttempts: 3,
	}, func(ctx context.Context) error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	var attempts int
	err := routine.Retry(context.Background(), routine.RetryConfig{
		MaxAttempts:     4,
		InitialInterval: 5 * time.Millisecond,
		BackoffFactor:   1.5,
		Jitter:          false,
	}, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetry_ExhaustedAttempts(t *testing.T) {
	expectedErr := errors.New("persistent outage")
	var attempts int

	err := routine.Retry(context.Background(), routine.RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 5 * time.Millisecond,
		BackoffFactor:   1.5,
	}, func(ctx context.Context) error {
		attempts++
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 3, attempts)
}

func TestRetry_RetryIfPredicate(t *testing.T) {
	errFatal := errors.New("fatal client error")
	var attempts int

	err := routine.Retry(context.Background(), routine.RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 5 * time.Millisecond,
		RetryIf: func(err error) bool {
			return !errors.Is(err, errFatal)
		},
	}, func(ctx context.Context) error {
		attempts++
		return errFatal
	})

	// Must abort immediately without retrying because RetryIf returned false
	assert.ErrorIs(t, err, errFatal)
	assert.Equal(t, 1, attempts)
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := routine.Retry(ctx, routine.RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 50 * time.Millisecond,
	}, func(ctx context.Context) error {
		return errors.New("temporary error")
	})

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGoWithRetry(t *testing.T) {
	var attempts int64
	done := make(chan struct{})

	routine.GoWithRetry(context.Background(), routine.RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 5 * time.Millisecond,
	}, func(ctx context.Context) error {
		curr := atomic.AddInt64(&attempts, 1)
		if curr < 2 {
			return errors.New("retry me")
		}
		close(done)
		return nil
	})

	select {
	case <-done:
		assert.Equal(t, int64(2), atomic.LoadInt64(&attempts))
	case <-time.After(1 * time.Second):
		t.Fatal("GoWithRetry timed out waiting for completion")
	}
}

func TestGoDetachedWithRetry(t *testing.T) {
	type key string
	const reqKey key = "request-id"

	reqCtx, cancel := context.WithCancel(context.WithValue(context.Background(), reqKey, "req-123"))
	cancel() // Cancel parent context immediately

	var retrievedVal string
	done := make(chan struct{})

	routine.GoDetachedWithRetry(reqCtx, routine.RetryConfig{
		MaxAttempts:     2,
		InitialInterval: 5 * time.Millisecond,
	}, func(bgCtx context.Context) error {
		if val, ok := bgCtx.Value(reqKey).(string); ok {
			retrievedVal = val
		}
		close(done)
		return nil
	})

	select {
	case <-done:
		assert.Equal(t, "req-123", retrievedVal)
	case <-time.After(1 * time.Second):
		t.Fatal("GoDetachedWithRetry timed out")
	}
}
