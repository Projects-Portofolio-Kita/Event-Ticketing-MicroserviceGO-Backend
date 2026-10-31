// Package queue provides concurrent background job queues and worker pools.
//
// File: worker_pool.go
// Usability: Defines Pool struct, constructor, capacity inspection metrics, and job submission.
package queue

import "sync"

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
	return &Pool{workerCount: workerCount, capacity: capacity, jobs: make(chan Job, capacity)}
}

// Submit queues a job for execution, returning ErrQueueFull if saturated
func (p *Pool) Submit(job Job) error {
	if job == nil { return ErrJobNil }
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.isClosed { return ErrQueueShutdown }
	select {
	case p.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

func (p *Pool) Capacity() int    { return p.capacity }
func (p *Pool) WorkerCount() int { return p.workerCount }
func (p *Pool) PendingJobs() int { return len(p.jobs) }