// Package queue provides concurrent background job queues and worker pools.
//
// File: worker_lifecycle.go
// Usability: Manages worker goroutine spawning, channel dispatching, and graceful pool draining.
package queue

import (
	"context"
	"fmt"
)

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
