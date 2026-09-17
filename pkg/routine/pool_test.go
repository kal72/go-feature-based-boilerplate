package routine_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/routine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_Execution(t *testing.T) {
	pool := routine.NewPool(4, routine.WithQueueSize(20))
	defer pool.Stop()

	assert.Equal(t, 4, pool.RunningWorkers())
	assert.Equal(t, 20, pool.QueueCapacity())

	var counter int64
	var wg sync.WaitGroup

	numTasks := 15
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		err := pool.Submit(context.Background(), func(ctx context.Context) {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		})
		require.NoError(t, err)
	}

	wg.Wait()
	assert.Equal(t, int64(numTasks), atomic.LoadInt64(&counter))
}

func TestPool_StrategyDiscard(t *testing.T) {
	// Pool with 1 worker and queue size of 1
	pool := routine.NewPool(1, routine.WithQueueSize(1), routine.WithStrategy(routine.StrategyDiscard))
	defer pool.Stop()

	blockWorker := make(chan struct{})
	workerStarted := make(chan struct{})

	// Task 1 occupies the 1 worker
	err := pool.Submit(context.Background(), func(ctx context.Context) {
		close(workerStarted)
		<-blockWorker
	})
	require.NoError(t, err)

	<-workerStarted

	// Task 2 fills the 1 queue slot
	err = pool.Submit(context.Background(), func(ctx context.Context) {})
	require.NoError(t, err)

	// Task 3 must be discarded because worker is busy and queue is full
	err = pool.Submit(context.Background(), func(ctx context.Context) {})
	assert.ErrorIs(t, err, routine.ErrQueueFull)

	close(blockWorker)
}

func TestPool_StrategyBlock(t *testing.T) {
	pool := routine.NewPool(1, routine.WithQueueSize(1), routine.WithStrategy(routine.StrategyBlock))
	defer pool.Stop()

	blockWorker := make(chan struct{})
	workerStarted := make(chan struct{})

	// Task 1 occupies the worker
	err := pool.Submit(context.Background(), func(ctx context.Context) {
		close(workerStarted)
		<-blockWorker
	})
	require.NoError(t, err)

	<-workerStarted

	// Task 2 fills the queue buffer
	err = pool.Submit(context.Background(), func(ctx context.Context) {})
	require.NoError(t, err)

	// Task 3 with canceled context should abort when blocking
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = pool.Submit(ctx, func(ctx context.Context) {})
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	close(blockWorker)
}

func TestPool_PanicRecovery(t *testing.T) {
	var panicCaught atomic.Bool
	routine.SetPanicHandler(func(ctx context.Context, r any, stack []byte) {
		if r == "worker-boom" {
			panicCaught.Store(true)
		}
	})

	pool := routine.NewPool(1, routine.WithQueueSize(5))
	defer pool.Stop()

	var nextTaskDone atomic.Bool
	done := make(chan struct{})

	// Submit panicking task
	err := pool.Submit(context.Background(), func(ctx context.Context) {
		panic("worker-boom")
	})
	require.NoError(t, err)

	// Submit subsequent task: worker must have recovered and execute it
	err = pool.Submit(context.Background(), func(ctx context.Context) {
		nextTaskDone.Store(true)
		close(done)
	})
	require.NoError(t, err)

	select {
	case <-done:
		assert.True(t, panicCaught.Load())
		assert.True(t, nextTaskDone.Load())
	case <-time.After(1 * time.Second):
		t.Fatal("subsequent task did not execute after worker panic")
	}
}

func TestPool_StopDrainsTasks(t *testing.T) {
	pool := routine.NewPool(2, routine.WithQueueSize(10))

	var executedCount int64
	numTasks := 6

	for i := 0; i < numTasks; i++ {
		err := pool.Submit(context.Background(), func(ctx context.Context) {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&executedCount, 1)
		})
		require.NoError(t, err)
	}

	// Stop blocks until all 6 submitted tasks are completed
	pool.Stop()
	assert.Equal(t, int64(numTasks), atomic.LoadInt64(&executedCount))

	// Subsequent submissions must fail with ErrPoolClosed
	err := pool.Submit(context.Background(), func(ctx context.Context) {})
	assert.ErrorIs(t, err, routine.ErrPoolClosed)
}
