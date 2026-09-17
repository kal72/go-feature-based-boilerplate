package routine_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/routine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contextKey string

func TestGo_Success(t *testing.T) {
	done := make(chan bool, 1)

	routine.Go(context.Background(), func(ctx context.Context) {
		done <- true
	})

	select {
	case val := <-done:
		assert.True(t, val)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for routine.Go")
	}
}

func TestGo_PanicRecovery(t *testing.T) {
	recoveredChan := make(chan any, 1)

	routine.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
		recoveredChan <- r
	})

	routine.Go(context.Background(), func(ctx context.Context) {
		panic("deliberate panic inside goroutine")
	})

	select {
	case val := <-recoveredChan:
		assert.Equal(t, "deliberate panic inside goroutine", val)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for panic recovery")
	}
}

func TestGoDetached_DecoupledAndValuesPreserved(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	ctxWithVal := context.WithValue(parentCtx, contextKey("trace_id"), "trace-xyz-123")

	// Cancel parent context immediately
	cancel()
	require.ErrorIs(t, ctxWithVal.Err(), context.Canceled)

	resultChan := make(chan struct {
		isCanceled bool
		traceID    any
	}, 1)

	routine.GoDetached(ctxWithVal, func(bgCtx context.Context) {
		resultChan <- struct {
			isCanceled bool
			traceID    any
		}{
			isCanceled: bgCtx.Err() != nil,
			traceID:    bgCtx.Value(contextKey("trace_id")),
		}
	})

	select {
	case res := <-resultChan:
		assert.False(t, res.isCanceled, "bgCtx should NOT be canceled when parent is canceled")
		assert.Equal(t, "trace-xyz-123", res.traceID, "context values must be preserved")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for GoDetached")
	}
}

func TestSafe_PanicRecovery(t *testing.T) {
	panicMsg := make(chan any, 1)
	routine.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
		panicMsg <- r
	})

	routine.Safe(func() {
		panic("safe panic test")
	})

	select {
	case msg := <-panicMsg:
		assert.Equal(t, "safe panic test", msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for routine.Safe panic handler")
	}
}

func TestWaitForShutdown_Success(t *testing.T) {
	var counter int64

	for i := 0; i < 5; i++ {
		routine.Go(context.Background(), func(ctx context.Context) {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt64(&counter, 1)
		})
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := routine.WaitForShutdown(shutdownCtx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), atomic.LoadInt64(&counter))
}

func TestWaitForShutdown_Timeout(t *testing.T) {
	// Goroutine that sleeps longer than shutdown timeout
	routine.Go(context.Background(), func(ctx context.Context) {
		time.Sleep(1 * time.Second)
	})

	// Shutdown timeout of only 20ms
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := routine.WaitForShutdown(shutdownCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown timed out")
}
