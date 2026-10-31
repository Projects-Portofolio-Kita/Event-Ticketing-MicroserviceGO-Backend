package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

var (
	// ErrQueueFull is returned when the job channel has reached capacity
	ErrQueueFull = errors.New("notification queue is full")

	// ErrQueueShutdown is returned when attempting to submit a job to a stopped worker pool
	ErrQueueShutdown = errors.New("worker pool is shutting down")

	// ErrJobNil is returned when a nil job is submitted
	ErrJobNil = errors.New("job cannot be nil")
)

// Job represents a unit of asynchronous work
type Job interface {
	Execute(ctx context.Context) error
	ID() string
}

// WorkerPool defines the contract for concurrent background processing
type WorkerPool interface {
	Start(ctx context.Context)
	Submit(job Job) error
	Stop(ctx context.Context) error
	Capacity() int
	WorkerCount() int
	PendingJobs() int
}

// FuncJob is an adapter that allows ordinary functions to be used as Job
type FuncJob struct {
	id   string
	exec func(ctx context.Context) error
}

// NewJob creates a Job from an ID and execution function
func NewJob(id string, exec func(ctx context.Context) error) Job {
	return &FuncJob{
		id:   id,
		exec: exec,
	}
}

// ID returns the job identifier
func (f *FuncJob) ID() string {
	return f.id
}

// Execute runs the underlying function
func (f *FuncJob) Execute(ctx context.Context) error {
	if f.exec == nil {
		return nil
	}
	return f.exec(ctx)
}

// Pool implements the WorkerPool interface using buffered channels and sync.WaitGroup
type Pool struct {
	workerCount int
	capacity    int
	jobs        chan Job
	wg          sync.WaitGroup
	mu          sync.RWMutex
	isClosed    bool
	startOnce   sync.Once
	stopOnce    sync.Once
}

// NewWorkerPool creates an initialized WorkerPool with the specified worker count and capacity
func NewWorkerPool(workerCount, capacity int) *Pool {
	if workerCount <= 0 {
		workerCount = 5
	}
	if capacity <= 0 {
		capacity = 500
	}

	return &Pool{
		workerCount: workerCount,
		capacity:    capacity,
		jobs:        make(chan Job, capacity),
	}
}

// Start launches the worker goroutines
func (p *Pool) Start(ctx context.Context) {
	p.startOnce.Do(func() {
		for i := 1; i <= p.workerCount; i++ {
			p.wg.Add(1)
			go p.worker(ctx, i)
		}
	})
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	for job := range p.jobs {
		p.executeJobSafely(ctx, id, job)
	}
}

func (p *Pool) executeJobSafely(ctx context.Context, id int, job Job) {
	defer func() {
		if r := recover(); r != nil {
			jobID := "unknown"
			if job != nil {
				jobID = job.ID()
			}
			log.Printf("[WorkerPool Worker %d] Job %s panicked: %v", id, jobID, r)
		}
	}()

	if job == nil {
		return
	}

	if err := job.Execute(ctx); err != nil {
		log.Printf("[WorkerPool Worker %d] Job %s execution failed: %v", id, job.ID(), err)
	}
}

// Submit queues a job for execution, returning ErrQueueFull if saturated
func (p *Pool) Submit(job Job) error {
	if job == nil {
		return ErrJobNil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.isClosed {
		return ErrQueueShutdown
	}

	select {
	case p.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// Stop initiates graceful shutdown, draining all queued jobs before exiting
func (p *Pool) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.isClosed = true
		close(p.jobs)
		p.mu.Unlock()
	})

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("worker pool graceful shutdown timed out: %w", ctx.Err())
	}
}

// Capacity returns the maximum buffered queue capacity
func (p *Pool) Capacity() int {
	return p.capacity
}

// WorkerCount returns the number of active workers
func (p *Pool) WorkerCount() int {
	return p.workerCount
}

// PendingJobs returns the number of jobs currently waiting in the buffer
func (p *Pool) PendingJobs() int {
	return len(p.jobs)
}
