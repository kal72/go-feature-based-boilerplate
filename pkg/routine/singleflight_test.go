package routine_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/routine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleflight_Deduplication(t *testing.T) {
	sf := routine.NewSingleflight[string]()

	var callCount int64
	var wg sync.WaitGroup

	numGoroutines := 30
	wg.Add(numGoroutines)

	results := make([]string, numGoroutines)
	sharedFlags := make([]bool, numGoroutines)

	// Barrier channel to release all goroutines simultaneously
	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-start

			val, err, shared := sf.Do(context.Background(), "user:profile:100", func(ctx context.Context) (string, error) {
				atomic.AddInt64(&callCount, 1)
				time.Sleep(30 * time.Millisecond) // Simulate DB query
				return "profile-data-100", nil
			})

			require.NoError(t, err)
			results[idx] = val
			sharedFlags[idx] = shared
		}()
	}

	close(start)
	wg.Wait()

	// The expensive function must have been called exactly ONCE despite 30 concurrent callers
	assert.Equal(t, int64(1), atomic.LoadInt64(&callCount))

	for i := 0; i < numGoroutines; i++ {
		assert.Equal(t, "profile-data-100", results[i])
		assert.True(t, sharedFlags[i], "all callers should indicate shared result")
	}
}

func TestSingleflight_DifferentKeys(t *testing.T) {
	sf := routine.NewSingleflight[int]()

	var counter int64

	val1, err1, _ := sf.Do(context.Background(), "key:1", func(ctx context.Context) (int, error) {
		atomic.AddInt64(&counter, 1)
		return 10, nil
	})
	val2, err2, _ := sf.Do(context.Background(), "key:2", func(ctx context.Context) (int, error) {
		atomic.AddInt64(&counter, 1)
		return 20, nil
	})

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 10, val1)
	assert.Equal(t, 20, val2)
	assert.Equal(t, int64(2), atomic.LoadInt64(&counter))
}

func TestSingleflight_DoChan(t *testing.T) {
	sf := routine.NewDeduplicator[string]()

	resChan := sf.DoChan(context.Background(), "async:key", func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "async-val", nil
	})

	select {
	case res := <-resChan:
		assert.NoError(t, res.Err)
		assert.Equal(t, "async-val", res.Val)
	case <-time.After(1 * time.Second):
		t.Fatal("DoChan timed out")
	}
}

func TestSingleflight_DoChan_ContextCancellation(t *testing.T) {
	sf := routine.NewSingleflight[string]()

	slowExecutionStarted := make(chan struct{})
	blockSlowExecution := make(chan struct{})

	// Goroutine 1: Leader with long context
	var valLeader string
	var errLeader error
	doneLeader := make(chan struct{})

	go func() {
		defer close(doneLeader)
		valLeader, errLeader, _ = sf.Do(context.Background(), "cancel:key", func(ctx context.Context) (string, error) {
			close(slowExecutionStarted)
			<-blockSlowExecution
			return "leader-finished", nil
		})
	}()

	<-slowExecutionStarted

	// Goroutine 2: Follower with fast timeout context
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	valFollower, errFollower, _ := sf.Do(ctxTimeout, "cancel:key", func(ctx context.Context) (string, error) {
		return "never-called", nil
	})

	// Follower should time out quickly
	assert.ErrorIs(t, errFollower, context.DeadlineExceeded)
	assert.Empty(t, valFollower)

	// Release leader
	close(blockSlowExecution)
	<-doneLeader

	// Leader must still succeed with its value
	assert.NoError(t, errLeader)
	assert.Equal(t, "leader-finished", valLeader)
}

func TestSingleflight_PanicRecovery(t *testing.T) {
	sf := routine.NewSingleflight[string]()

	var panicCaught atomic.Bool
	routine.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
		if fmt.Sprintf("%v", r) == "sf-boom" {
			panicCaught.Store(true)
		}
	})

	numWaiters := 5
	var wg sync.WaitGroup
	wg.Add(numWaiters)

	start := make(chan struct{})
	errs := make([]error, numWaiters)

	for i := 0; i < numWaiters; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-start
			_, err, _ := sf.Do(context.Background(), "panic:key", func(ctx context.Context) (string, error) {
				time.Sleep(10 * time.Millisecond)
				panic("sf-boom")
			})
			errs[idx] = err
		}()
	}

	close(start)
	wg.Wait()

	assert.True(t, panicCaught.Load())
	for i := 0; i < numWaiters; i++ {
		assert.Error(t, errs[i])
		assert.Contains(t, errs[i].Error(), "panic recovered: sf-boom")
	}
}

func TestSingleflight_Forget(t *testing.T) {
	sf := routine.NewSingleflight[int]()

	var counter int64

	runFn := func(ctx context.Context) (int, error) {
		curr := atomic.AddInt64(&counter, 1)
		return int(curr), nil
	}

	val1, err1, _ := sf.Do(context.Background(), "forget:key", runFn)
	require.NoError(t, err1)
	assert.Equal(t, 1, val1)

	// Explicitly forget key
	sf.Forget("forget:key")

	// Next call should execute anew
	val2, err2, _ := sf.Do(context.Background(), "forget:key", runFn)
	require.NoError(t, err2)
	assert.Equal(t, 2, val2)
}

func TestSingleflight_ErrorPropagation(t *testing.T) {
	sf := routine.NewSingleflight[string]()

	expectedErr := errors.New("db error")
	val, err, _ := sf.Do(context.Background(), "err:key", func(ctx context.Context) (string, error) {
		return "", expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	assert.Empty(t, val)
}
