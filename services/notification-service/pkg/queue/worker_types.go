// Package queue provides concurrent background job queues and worker pools.
//
// File: worker_types.go
// Usability: Declares Job and WorkerPool interfaces, queue error definitions, and FuncJob function adapter.
package queue

import (
	"context"
	"errors"
)

var (
	ErrQueueFull     = errors.New("notification queue is full")
	ErrQueueShutdown = errors.New("worker pool is shutting down")
	ErrJobNil        = errors.New("job cannot be nil")
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
	return &FuncJob{id: id, exec: exec}
}

func (f *FuncJob) ID() string { return f.id }

func (f *FuncJob) Execute(ctx context.Context) error {
	if f.exec == nil { return nil }
	return f.exec(ctx)
}