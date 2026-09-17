package routine

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// Result represents the outcome of a singleflight execution.
type Result[T any] struct {
	// Val contains the computed value on successful execution.
	Val T

	// Err contains the error returned by the execution, or panic recovery error.
	Err error

	// Shared indicates whether this result was shared with concurrent callers.
	Shared bool
}

// call tracks an in-flight or completed execution for a given key.
type call[T any] struct {
	wg    sync.WaitGroup
	val   T
	err   error
	dups  int
	chans []chan Result[T]
}

// Singleflight coordinates concurrent duplicate function calls, suppressing redundant
// executions for the same key and sharing the computed result across all concurrent callers.
//
// This is the premier pattern for eliminating Cache Stampede / Thundering Herd when
// heavily accessed cache keys expire or when fetching expensive database aggregations.
//
// Type safety is guaranteed via Go Generics ([T any]), removing the need for manual
// type assertion.
type Singleflight[T any] struct {
	mu sync.Mutex
	m  map[string]*call[T]
}

// Deduplicator is an alias for Singleflight.
type Deduplicator[T any] = Singleflight[T]

// NewSingleflight initializes and returns a new Singleflight coordinator.
//
// Example usage:
//
//	sf := routine.NewSingleflight[*entity.User]()
//	user, err, shared := sf.Do(ctx, fmt.Sprintf("user:%d", id), func(ctx context.Context) (*entity.User, error) {
//	    return userRepo.FindByID(ctx, id)
//	})
func NewSingleflight[T any]() *Singleflight[T] {
	return &Singleflight[T]{
		m: make(map[string]*call[T]),
	}
}

// NewDeduplicator initializes a new Deduplicator coordinator (alias of NewSingleflight).
func NewDeduplicator[T any]() *Deduplicator[T] {
	return NewSingleflight[T]()
}

// Do executes and returns the result of the given function, ensuring that only one execution
// is in-flight for a given key at a time.
//
// If duplicate calls arrive while an execution is in-flight:
//  1. The duplicate callers wait for the original execution to finish.
//  2. The duplicate callers receive the same computed value, error, and shared = true.
//  3. If a waiting caller's context is canceled or times out before completion, it exits
//     immediately with ctx.Err() without interrupting the in-flight execution.
//  4. If fn panics, the panic is safely recovered, logged via PanicHandler, recorded
//     in metrics, and returned as an error to all callers without crashing the server.
func (g *Singleflight[T]) Do(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, error, bool) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		c.dups++
		ch := make(chan Result[T], 1)
		c.chans = append(c.chans, ch)
		g.mu.Unlock()

		select {
		case r := <-ch:
			return r.Val, r.Err, true
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err(), true
		}
	}

	c := &call[T]{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	g.execute(ctx, key, c, fn)

	return c.val, c.err, c.dups > 0
}

// DoChan executes fn asynchronously, returning a channel that receives the Result when ready.
//
// This allows callers to select on completion alongside other channel events or independent
// cancellation deadlines without blocking a thread.
//
// Example usage:
//
//	resChan := sf.DoChan(ctx, key, fn)
//	select {
//	case r := <-resChan:
//	    return r.Val, r.Err
//	case <-ctx.Done():
//	    return nil, ctx.Err()
//	}
func (g *Singleflight[T]) DoChan(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) <-chan Result[T] {
	ch := make(chan Result[T], 1)

	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		c.dups++
		c.chans = append(c.chans, ch)
		g.mu.Unlock()
		return ch
	}

	c := &call[T]{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	go func() {
		g.execute(ctx, key, c, fn)
		ch <- Result[T]{
			Val:    c.val,
			Err:    c.err,
			Shared: c.dups > 0,
		}
	}()

	return ch
}

// Forget removes the specified key from the active deduplication map.
// Subsequent calls for key will start a new execution instead of waiting on the prior one.
func (g *Singleflight[T]) Forget(key string) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}

// execute runs fn, protects against panics, updates metrics, and dispatches results to waiting channels.
func (g *Singleflight[T]) execute(ctx context.Context, key string, c *call[T], fn func(ctx context.Context) (T, error)) {
	start := time.Now()
	metricIncActive()

	defer func() {
		duration := time.Since(start).Seconds()
		metricDecActive()
		metricIncTasks("singleflight")
		metricObserveDuration("singleflight", duration)

		if r := recover(); r != nil {
			metricIncPanics()
			stack := debug.Stack()
			getPanicHandler()(ctx, r, stack)

			var zero T
			c.val = zero
			c.err = fmt.Errorf("routine.Singleflight: panic recovered: %v", r)
		}

		g.mu.Lock()
		delete(g.m, key)
		chans := c.chans
		c.chans = nil
		g.mu.Unlock()

		res := Result[T]{
			Val:    c.val,
			Err:    c.err,
			Shared: true,
		}
		for _, ch := range chans {
			ch <- res
		}

		c.wg.Done()
	}()

	c.val, c.err = fn(ctx)
}
