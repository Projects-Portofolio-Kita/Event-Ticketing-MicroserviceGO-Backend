// Package queue provides concurrent background job queues and worker pools.
//
// File: worker_exec.go
// Usability: Implements panic-safe and error-logged execution wrappers for background jobs.
package queue

import (
	"context"
	"log"
)

func (p *Pool) executeJobSafely(ctx context.Context, id int, job Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WorkerPool Worker %d] Job panicked: %v", id, r)
		}
	}()
	if job != nil {
		if err := job.Execute(ctx); err != nil {
			log.Printf("[WorkerPool Worker %d] Job %s failed: %v", id, job.ID(), err)
		}
	}
}
