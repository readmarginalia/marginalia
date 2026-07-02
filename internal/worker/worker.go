package worker

import (
	"context"
	"errors"
	"log/slog"
	"marginalia/internal/correlation"
	"sync"
	"time"
)

type JobHandler func(context.Context) error

var ErrWorkerStopped = errors.New("worker stopped")

const globalMaxAttempts = 10

type Job struct {
	Name          string
	Handler       JobHandler
	MaxAttempts   int
	CorrelationId string
}

type WorkerPool struct {
	jobs     chan Job
	poolSize int

	cancel context.CancelFunc

	wg       sync.WaitGroup
	mu       sync.RWMutex
	started  bool
	stopping bool
}

func NewWorkerPool(poolSize int, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		jobs:     make(chan Job, queueSize),
		poolSize: poolSize,
	}

	return wp
}

func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.stopping {
		return ErrWorkerStopped
	}

	if wp.started {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	wp.cancel = cancel
	wp.started = true

	for range wp.poolSize {
		wp.wg.Go(func() {
			wp.workerLoop(runCtx)
		})
	}

	return nil
}

func (wp *WorkerPool) Enqueue(ctx context.Context, name string, handler JobHandler, maxAttempts int) error {
	if handler == nil {
		return errors.New("job handler cannot be nil")
	}

	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	if maxAttempts > globalMaxAttempts {
		maxAttempts = globalMaxAttempts
	}

	correlationId, _ := correlation.FromContext(ctx)

	job := Job{
		Name:          name,
		Handler:       handler,
		MaxAttempts:   maxAttempts,
		CorrelationId: correlationId,
	}

	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if wp.stopping || !wp.started {
		return ErrWorkerStopped
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.jobs <- job:
		slog.InfoContext(ctx, "job enqueued",
			"job_name", job.Name,
			"max_attempts", job.MaxAttempts,
		)
		return nil
	}
}

func (wp *WorkerPool) Shutdown(ctx context.Context) error {
	wp.mu.Lock()

	if !wp.stopping {
		wp.stopping = true
		close(wp.jobs)
	}

	wp.mu.Unlock()

	done := make(chan struct{})

	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		if wp.cancel != nil {
			wp.cancel()
		}
		return ctx.Err()
	}
}

func (wp *WorkerPool) workerLoop(ctx context.Context) {
	for job := range wp.jobs {
		wp.execute(ctx, job)
	}
}

func (wp *WorkerPool) execute(ctx context.Context, job Job) {
	ctx = wp.jobContext(ctx, job)
	for attempt := 1; attempt <= job.MaxAttempts; attempt++ {
		start := time.Now()
		err := job.Handler(ctx)
		duration := time.Since(start)

		if err == nil {
			slog.InfoContext(ctx, "job completed successfully",
				"job_name", job.Name,
				"attempt", attempt,
				"max_attempts", job.MaxAttempts,
				"duration_ms", duration.Milliseconds(),
			)
			return
		}

		if attempt == job.MaxAttempts {
			slog.ErrorContext(ctx,
				"job max attempts reached",
				"job_name", job.Name,
				"attempt", attempt,
				"max_attempts", job.MaxAttempts,
				"error", err,
				"duration_ms", duration.Milliseconds(),
			)
			return
		}

		delay := retryDelay(attempt)
		slog.WarnContext(ctx,
			"job failed, retrying",
			"job_name", job.Name,
			"attempt", attempt,
			"max_attempts", job.MaxAttempts,
			"retry_delay", delay,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func retryDelay(attempt int) time.Duration {
	const baseDelay = 500 * time.Millisecond
	const maxDelay = 10 * time.Second

	if attempt <= 1 {
		return baseDelay
	}

	delay := baseDelay * time.Duration(1<<(attempt-1))

	if delay > maxDelay {
		return maxDelay
	}

	return delay
}

func (wp *WorkerPool) jobContext(ctx context.Context, job Job) context.Context {
	if job.CorrelationId != "" {
		ctx = correlation.NewContext(ctx, job.CorrelationId)
	}

	return ctx
}
