package routine

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// Option configures Group behavior.
type Option func(*Group)

// WithLimit restricts the maximum number of active concurrent goroutines
// executing inside the Group to limit. Values <= 0 indicate unlimited concurrency.
func WithLimit(limit int) Option {
	return func(g *Group) {
		if limit > 0 {
			g.sem = make(chan struct{}, limit)
		}
	}
}

// WithTimeout sets a maximum total execution deadline for all tasks within the Group.
func WithTimeout(timeout time.Duration) Option {
	return func(g *Group) {
		if timeout > 0 {
			g.ctx, g.cancel = context.WithTimeout(g.ctx, timeout)
		}
	}
}

// Group coordinates concurrent task execution (Fan-Out / Fan-In) with panic recovery,
// optional concurrency rate-limiting, and error-driven sibling cancellation.
type Group struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	sem     chan struct{}
	errOnce sync.Once
	err     error
}

// NewGroup initializes a new concurrent task Group derived from parent ctx.
//
// Example usage:
//
//	g := routine.NewGroup(ctx, routine.WithLimit(5), routine.WithTimeout(3*time.Second))
//	g.Submit(func(ctx context.Context) error { return fetchA(ctx) })
//	g.Submit(func(ctx context.Context) error { return fetchB(ctx) })
//	if err := g.Wait(); err != nil {
//	    return err
//	}
func NewGroup(ctx context.Context, opts ...Option) *Group {
	groupCtx, cancel := context.WithCancel(ctx)
	g := &Group{
		ctx:    groupCtx,
		cancel: cancel,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Context returns the context associated with the Group.
// The context is canceled as soon as the first task returns a non-nil error,
// if a panic occurs, or when Wait completes.
func (g *Group) Context() context.Context {
	return g.ctx
}

// Submit schedules a function to be executed concurrently in a managed goroutine.
//
// Behavior:
//  1. Concurrency Limiting: If WithLimit was set, Submit blocks until a worker slot is free.
//  2. Cancellation Check: If the Group context is already canceled (due to a previous error
//     or timeout), Submit exits early and returns ctx.Err().
//  3. Panic Recovery: If fn panics, the panic is intercepted and converted into a formatted
//     error, preventing process termination and triggering sibling cancellation.
//  4. Error Propagation: The first non-nil error returned by any task cancels the Group
//     context and is returned by Wait.
func (g *Group) Submit(fn func(ctx context.Context) error) {
	g.wg.Add(1)

	// Acquire semaphore slot if limit is configured
	if g.sem != nil {
		select {
		case g.sem <- struct{}{}:
		case <-g.ctx.Done():
			g.wg.Done()
			return
		}
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				getPanicHandler()(g.ctx, r, stack)
				g.recordError(fmt.Errorf("routine.Group: panic recovered: %v", r))
			}

			if g.sem != nil {
				<-g.sem
			}
			g.wg.Done()
		}()

		// Do not execute if context is already terminated
		if err := g.ctx.Err(); err != nil {
			return
		}

		if err := fn(g.ctx); err != nil {
			g.recordError(err)
		}
	}()
}

// recordError stores the first error encountered and cancels the Group context.
func (g *Group) recordError(err error) {
	if err == nil {
		return
	}
	g.errOnce.Do(func() {
		g.err = err
		if g.cancel != nil {
			g.cancel()
		}
	})
}

// Wait blocks until all submitted tasks have completed, releases all resources,
// and returns the first non-nil error encountered (if any).
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}
