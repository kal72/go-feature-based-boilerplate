package routine_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/routine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroup_Success(t *testing.T) {
	ctx := context.Background()
	g := routine.NewGroup(ctx)

	var counter int64

	for i := 0; i < 10; i++ {
		g.Submit(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
	}

	err := g.Wait()
	assert.NoError(t, err)
	assert.Equal(t, int64(10), atomic.LoadInt64(&counter))
}

func TestGroup_ConcurrencyLimit(t *testing.T) {
	ctx := context.Background()
	limit := 3
	g := routine.NewGroup(ctx, routine.WithLimit(limit))

	var currentConcurrent int64
	var maxConcurrent int64

	for i := 0; i < 12; i++ {
		g.Submit(func(ctx context.Context) error {
			curr := atomic.AddInt64(&currentConcurrent, 1)
			defer atomic.AddInt64(&currentConcurrent, -1)

			// Track maximum concurrency observed
			for {
				max := atomic.LoadInt64(&maxConcurrent)
				if curr <= max || atomic.CompareAndSwapInt64(&maxConcurrent, max, curr) {
					break
				}
			}

			time.Sleep(30 * time.Millisecond)
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.LessOrEqual(t, atomic.LoadInt64(&maxConcurrent), int64(limit), "concurrency must not exceed configured limit")
}

func TestGroup_ErrorPropagationAndCancellation(t *testing.T) {
	ctx := context.Background()
	g := routine.NewGroup(ctx)

	expectedErr := errors.New("upstream service failed")
	canceledCount := int64(0)

	g.Submit(func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return expectedErr
	})

	for i := 0; i < 5; i++ {
		g.Submit(func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				return nil
			case <-ctx.Done():
				atomic.AddInt64(&canceledCount, 1)
				return ctx.Err()
			}
		})
	}

	err := g.Wait()
	assert.ErrorIs(t, err, expectedErr)
	assert.Greater(t, atomic.LoadInt64(&canceledCount), int64(0), "sibling tasks should be canceled upon first error")
}

func TestGroup_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	g := routine.NewGroup(ctx)

	g.Submit(func(ctx context.Context) error {
		panic("boom in group task")
	})

	err := g.Wait()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom in group task")
}

func TestGroup_Timeout(t *testing.T) {
	ctx := context.Background()
	g := routine.NewGroup(ctx, routine.WithTimeout(50*time.Millisecond))

	g.Submit(func(ctx context.Context) error {
		select {
		case <-time.After(500 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	err := g.Wait()
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
