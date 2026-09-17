package routine

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Strategy defines backpressure handling when the worker pool queue is full.
type Strategy int

const (
	// StrategyBlock blocks the caller until a queue slot becomes available or ctx is canceled.
	StrategyBlock Strategy = iota

	// StrategyDiscard immediately returns ErrQueueFull if the queue is full.
	StrategyDiscard
)

var (
	// ErrQueueFull is returned when submitting to a full pool with StrategyDiscard.
	ErrQueueFull = errors.New("routine: worker pool queue is full")

	// ErrPoolClosed is returned when attempting to submit a task to a stopped pool.
	ErrPoolClosed = errors.New("routine: worker pool is closed")
)

// PoolOption configures Pool behavior.
type PoolOption func(*Pool)

// WithQueueSize sets the internal task queue buffer capacity.
// If size <= 0, a default capacity of 100 is used.
func WithQueueSize(size int) PoolOption {
	return func(p *Pool) {
		if size > 0 {
			p.queueSize = size
		}
	}
}

// WithStrategy sets the backpressure strategy (StrategyBlock or StrategyDiscard).
func WithStrategy(strategy Strategy) PoolOption {
	return func(p *Pool) {
		p.strategy = strategy
	}
}

type poolTask struct {
	ctx context.Context
	fn  func(ctx context.Context)
}

// Pool represents a bounded worker pool that processes submitted tasks across
// a fixed set of persistent worker goroutines with backpressure and panic safety.
type Pool struct {
	workerCount int
	queueSize   int
	strategy    Strategy
	tasks       chan poolTask
	wg          sync.WaitGroup
	closed      atomic.Bool
	stopOnce    sync.Once
}

// NewPool initializes and starts a new worker pool with workerCount goroutines.
//
// Defaults:
// - queueSize: 100
// - strategy: StrategyBlock
//
// Example usage:
//
//	pool := routine.NewPool(4, routine.WithQueueSize(50), routine.WithStrategy(routine.StrategyBlock))
//	defer pool.Stop()
//
//	err := pool.Submit(ctx, func(ctx context.Context) {
//	    processOrder(ctx)
//	})
func NewPool(workerCount int, opts ...PoolOption) *Pool {
	if workerCount <= 0 {
		workerCount = 1
	}

	p := &Pool{
		workerCount: workerCount,
		queueSize:   100,
		strategy:    StrategyBlock,
	}

	for _, opt := range opts {
		opt(p)
	}

	p.tasks = make(chan poolTask, p.queueSize)

	// Launch worker goroutines
	p.wg.Add(p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		go p.workerLoop()
	}

	return p
}

// workerLoop runs continuously on each worker goroutine, executing tasks until the queue is closed.
func (p *Pool) workerLoop() {
	defer p.wg.Done()

	for task := range p.tasks {
		p.executeTask(task)
	}
}

// executeTask executes a single task with panic recovery and telemetry observation.
func (p *Pool) executeTask(task poolTask) {
	metricIncActive()
	start := time.Now()

	defer func() {
		duration := time.Since(start).Seconds()
		metricDecActive()
		metricIncTasks("pool")
		metricObserveDuration("pool", duration)

		if r := recover(); r != nil {
			metricIncPanics()
			stack := debug.Stack()
			getPanicHandler()(task.ctx, r, stack)
		}
	}()

	task.fn(task.ctx)
}

// Submit enqueues a task for execution by one of the pool's workers.
//
// Behavior:
// 1. If the pool is stopped, returns ErrPoolClosed.
// 2. If ctx is already canceled, returns ctx.Err().
// 3. Under StrategyBlock: blocks until queue space is available or ctx is canceled.
// 4. Under StrategyDiscard: immediately returns ErrQueueFull if queue buffer is full.
func (p *Pool) Submit(ctx context.Context, fn func(ctx context.Context)) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	task := poolTask{
		ctx: ctx,
		fn:  fn,
	}

	switch p.strategy {
	case StrategyDiscard:
		select {
		case p.tasks <- task:
			return nil
		default:
			return ErrQueueFull
		}

	case StrategyBlock:
		fallthrough
	default:
		select {
		case p.tasks <- task:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Stop gracefully shuts down the worker pool. It closes the incoming task queue,
// prevents new submissions, and blocks until all queued tasks have finished executing.
// Safe to call multiple times.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		p.closed.Store(true)
		close(p.tasks)
	})
	p.wg.Wait()
}

// RunningWorkers returns the configured number of worker goroutines in the pool.
func (p *Pool) RunningWorkers() int {
	return p.workerCount
}

// QueueCapacity returns the configured buffer capacity of the task queue.
func (p *Pool) QueueCapacity() int {
	return p.queueSize
}
