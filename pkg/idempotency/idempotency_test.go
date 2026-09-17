package idempotency_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/idempotency"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type sampleOrder struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
}

func TestExecutor_SuccessAndCacheHit(t *testing.T) {
	storage := idempotency.NewMemoryStorage()
	exec := idempotency.NewExecutor[*sampleOrder](storage)

	var callCount int64
	runFn := func(ctx context.Context) (*sampleOrder, error) {
		atomic.AddInt64(&callCount, 1)
		return &sampleOrder{ID: "order-101", Amount: 250.50}, nil
	}

	key := "idemp-key-1"

	// First execution: should execute fn and cache response
	order1, err := exec.Execute(context.Background(), key, runFn)
	require.NoError(t, err)
	assert.Equal(t, "order-101", order1.ID)
	assert.Equal(t, 250.50, order1.Amount)
	assert.Equal(t, int64(1), atomic.LoadInt64(&callCount))

	// Second execution with SAME key: should return cached order WITHOUT calling fn again
	order2, err := exec.Execute(context.Background(), key, runFn)
	require.NoError(t, err)
	assert.Equal(t, "order-101", order2.ID)
	assert.Equal(t, 250.50, order2.Amount)
	assert.Equal(t, int64(1), atomic.LoadInt64(&callCount), "fn should NOT be invoked again")
}

func TestExecutor_EmptyKeyPassThrough(t *testing.T) {
	storage := idempotency.NewMemoryStorage()
	exec := idempotency.NewExecutor[string](storage)

	var callCount int64
	runFn := func(ctx context.Context) (string, error) {
		atomic.AddInt64(&callCount, 1)
		return "result", nil
	}

	// Empty key: executes fn every time without caching
	res1, err := exec.Execute(context.Background(), "", runFn)
	require.NoError(t, err)
	assert.Equal(t, "result", res1)

	res2, err := exec.Execute(context.Background(), "", runFn)
	require.NoError(t, err)
	assert.Equal(t, "result", res2)

	assert.Equal(t, int64(2), atomic.LoadInt64(&callCount))
}

func TestExecutor_ConcurrentConflict(t *testing.T) {
	storage := idempotency.NewMemoryStorage()
	exec := idempotency.NewExecutor[string](storage, idempotency.WithLockTTL(5*time.Second))

	key := "concurrent-key"
	executionStarted := make(chan struct{})
	blockExecution := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	var resLeader string
	var errLeader error

	var resFollower string
	var errFollower error

	// Request 1 (Leader): holds execution
	go func() {
		defer wg.Done()
		resLeader, errLeader = exec.Execute(context.Background(), key, func(ctx context.Context) (string, error) {
			close(executionStarted)
			<-blockExecution
			return "leader-data", nil
		})
	}()

	<-executionStarted

	// Request 2 (Follower): tries to execute concurrently with same key
	go func() {
		defer wg.Done()
		resFollower, errFollower = exec.Execute(context.Background(), key, func(ctx context.Context) (string, error) {
			return "follower-data", nil
		})
	}()

	// Wait briefly then unblock leader
	time.Sleep(20 * time.Millisecond)
	close(blockExecution)
	wg.Wait()

	// Leader must succeed
	assert.NoError(t, errLeader)
	assert.Equal(t, "leader-data", resLeader)

	// Follower must be rejected with ErrConcurrentRequest
	assert.ErrorIs(t, errFollower, idempotency.ErrConcurrentRequest)
	assert.Empty(t, resFollower)
}

func TestExecutor_ErrorAllowsRetry(t *testing.T) {
	storage := idempotency.NewMemoryStorage()
	exec := idempotency.NewExecutor[string](storage)

	key := "error-retry-key"
	var attempts int64
	transientErr := errors.New("temporary failure")

	runFn := func(ctx context.Context) (string, error) {
		curr := atomic.AddInt64(&attempts, 1)
		if curr == 1 {
			return "", transientErr
		}
		return "success-after-fix", nil
	}

	// First attempt: fails
	_, err := exec.Execute(context.Background(), key, runFn)
	assert.ErrorIs(t, err, transientErr)

	// Second attempt with same key: key was cleaned up, so retry is allowed
	res, err := exec.Execute(context.Background(), key, runFn)
	require.NoError(t, err)
	assert.Equal(t, "success-after-fix", res)
	assert.Equal(t, int64(2), atomic.LoadInt64(&attempts))
}

func TestFromContext(t *testing.T) {
	// Empty context
	assert.Empty(t, idempotency.FromContext(context.Background()))

	// Context with idempotency-key
	ctx1 := metadata.NewIncomingContext(context.Background(), metadata.Pairs("idempotency-key", "key-abc"))
	assert.Equal(t, "key-abc", idempotency.FromContext(ctx1))

	// Context with x-idempotency-key
	ctx2 := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-idempotency-key", "key-xyz"))
	assert.Equal(t, "key-xyz", idempotency.FromContext(ctx2))
}

func TestMemoryStorage_TTLExpiration(t *testing.T) {
	storage := idempotency.NewMemoryStorage()
	key := "ttl-test-key"

	// Lock with short TTL
	acquired, err := storage.Lock(context.Background(), key, 50*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Immediate re-lock should fail
	acquired2, err := storage.Lock(context.Background(), key, 50*time.Millisecond)
	require.NoError(t, err)
	assert.False(t, acquired2)

	// Wait for TTL to expire
	time.Sleep(70 * time.Millisecond)

	// Re-lock should now succeed
	acquired3, err := storage.Lock(context.Background(), key, 50*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, acquired3)
}
