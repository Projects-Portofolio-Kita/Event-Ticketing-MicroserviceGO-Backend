package queue_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
)

// TestWorkerPool_Stress1000Jobs verifies high-throughput concurrent submission of 1000 jobs.
func TestWorkerPool_Stress1000Jobs(t *testing.T) {
	const numJobs = 1000
	const workerCount = 10
	const queueCap = 1500

	wp := queue.NewWorkerPool(workerCount, queueCap)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	var executedCount atomic.Int32
	var submitErrors atomic.Int32
	var wg sync.WaitGroup
	wg.Add(numJobs)

	startTime := time.Now()

	// Launch 100 goroutines, each submitting 10 jobs (total 1000 concurrent submissions)
	const concurrentClients = 100
	const jobsPerClient = numJobs / concurrentClients

	var clientWg sync.WaitGroup
	clientWg.Add(concurrentClients)

	for c := 0; c < concurrentClients; c++ {
		go func(clientID int) {
			defer clientWg.Done()
			for j := 0; j < jobsPerClient; j++ {
				jobID := fmt.Sprintf("job-%d-%d", clientID, j)
				job := queue.NewJob(jobID, func(jCtx context.Context) error {
					defer wg.Done()
					executedCount.Add(1)
					time.Sleep(100 * time.Microsecond)
					return nil
				})

				err := wp.Submit(job)
				if err != nil {
					submitErrors.Add(1)
					wg.Done() // compensate wg
				}
			}
		}(c)
	}

	clientWg.Wait()
	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("Processed %d jobs in %v with %d workers (Submit errors: %d)",
		executedCount.Load(), elapsed, workerCount, submitErrors.Load())

	if submitErrors.Load() > 0 {
		t.Fatalf("encountered %d submit errors in adequately sized queue", submitErrors.Load())
	}

	if executedCount.Load() != numJobs {
		t.Fatalf("expected %d executed jobs, got %d", numJobs, executedCount.Load())
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("failed to stop worker pool: %v", err)
	}
}

// TestWorkerPool_QueueSaturationGracefulRejection verifies that when the queue buffer is full,
// Submit rejects immediately with ErrQueueFull and does not block or deadlock.
func TestWorkerPool_QueueSaturationGracefulRejection(t *testing.T) {
	const capacity = 50
	const workerCount = 2
	wp := queue.NewWorkerPool(workerCount, capacity)

	// Hold workers with a blocker channel so the queue fills up completely
	workerHold := make(chan struct{})
	wp.Start(context.Background())

	// Enqueue 2 blocker jobs to occupy the workers
	for i := 0; i < workerCount; i++ {
		job := queue.NewJob(fmt.Sprintf("blocker-%d", i), func(ctx context.Context) error {
			<-workerHold
			return nil
		})
		if err := wp.Submit(job); err != nil {
			t.Fatalf("failed to submit initial blocker job: %v", err)
		}
	}

	// Give a few ms for workers to pick up the 2 blocker jobs
	time.Sleep(10 * time.Millisecond)

	// Now fill the queue channel to capacity (50)
	for i := 0; i < capacity; i++ {
		job := queue.NewJob(fmt.Sprintf("buffered-%d", i), func(ctx context.Context) error {
			return nil
		})
		if err := wp.Submit(job); err != nil {
			t.Fatalf("failed to submit job %d into buffer: %v", i, err)
		}
	}

	// Queue is now completely saturated.
	// Submitting 200 more jobs must immediately fail with ErrQueueFull, without blocking!
	const overflowCount = 200
	var rejectedCount atomic.Int32
	var blockedGoroutines atomic.Int32

	var overflowWg sync.WaitGroup
	overflowWg.Add(overflowCount)

	startOverflow := time.Now()
	for i := 0; i < overflowCount; i++ {
		go func(idx int) {
			defer overflowWg.Done()
			job := queue.NewJob(fmt.Sprintf("overflow-%d", idx), func(ctx context.Context) error {
				return nil
			})

			done := make(chan struct{})
			go func() {
				err := wp.Submit(job)
				if errors.Is(err, queue.ErrQueueFull) {
					rejectedCount.Add(1)
				}
				close(done)
			}()

			select {
			case <-done:
				// Fast rejection
			case <-time.After(500 * time.Millisecond):
				blockedGoroutines.Add(1)
			}
		}(i)
	}

	overflowWg.Wait()
	overflowDuration := time.Since(startOverflow)

	t.Logf("Overflow submissions (%d): %d rejected with ErrQueueFull, %d blocked, elapsed %v",
		overflowCount, rejectedCount.Load(), blockedGoroutines.Load(), overflowDuration)

	if blockedGoroutines.Load() > 0 {
		t.Fatalf("%d goroutines were blocked during queue saturation!", blockedGoroutines.Load())
	}

	if rejectedCount.Load() != overflowCount {
		t.Fatalf("expected all %d overflow jobs to be rejected with ErrQueueFull, but got %d",
			overflowCount, rejectedCount.Load())
	}

	// Release workers
	close(workerHold)

	// Draining phase
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("failed to stop pool: %v", err)
	}
}

// TestWorkerPool_GracefulShutdownDrainingCompleteness stress-tests that Stop() drains all
// buffered jobs without dropping any, even under heavy load.
func TestWorkerPool_GracefulShutdownDrainingCompleteness(t *testing.T) {
	const workerCount = 4
	const queueCap = 200
	const numJobs = 150

	wp := queue.NewWorkerPool(workerCount, queueCap)

	var executedCount atomic.Int32

	// Pre-fill the queue BEFORE starting workers
	for i := 0; i < numJobs; i++ {
		job := queue.NewJob(fmt.Sprintf("drain-%d", i), func(ctx context.Context) error {
			time.Sleep(1 * time.Millisecond) // simulate non-trivial work
			executedCount.Add(1)
			return nil
		})
		if err := wp.Submit(job); err != nil {
			t.Fatalf("failed to submit drain job %d: %v", i, err)
		}
	}

	if wp.PendingJobs() != numJobs {
		t.Fatalf("expected %d pending jobs in buffer, got %d", numJobs, wp.PendingJobs())
	}

	// Start workers and immediately invoke Stop
	wp.Start(context.Background())

	stopStart := time.Now()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	drainDuration := time.Since(stopStart)
	t.Logf("Drained %d buffered jobs across %d workers in %v (Zero loss)",
		executedCount.Load(), workerCount, drainDuration)

	if executedCount.Load() != numJobs {
		t.Fatalf("Data loss during shutdown! Expected %d executed jobs, got %d", numJobs, executedCount.Load())
	}

	if wp.PendingJobs() != 0 {
		t.Fatalf("expected 0 pending jobs after Stop(), got %d", wp.PendingJobs())
	}
}

// TestWorkerPool_ConcurrentSubmitAndStopRace tests for race conditions or panics
// when multiple goroutines call Submit while Stop is called concurrently.
func TestWorkerPool_ConcurrentSubmitAndStopRace(t *testing.T) {
	for attempt := 0; attempt < 10; attempt++ {
		wp := queue.NewWorkerPool(4, 100)
		wp.Start(context.Background())

		var wg sync.WaitGroup
		const submitters = 20
		stopSignal := make(chan struct{})

		for s := 0; s < submitters; s++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-stopSignal // wait for simultaneous start
				for i := 0; i < 50; i++ {
					job := queue.NewJob(fmt.Sprintf("race-%d-%d", id, i), func(ctx context.Context) error {
						return nil
					})
					// This should either succeed, return ErrQueueFull, or return ErrQueueShutdown.
					// IT MUST NEVER PANIC (e.g., send on closed channel)!
					_ = wp.Submit(job)
				}
			}(s)
		}

		// Trigger concurrent submitters and concurrent Stop
		close(stopSignal)
		time.Sleep(100 * time.Microsecond) // tiny offset to maximize race collision probability

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = wp.Stop(stopCtx)
		stopCancel()

		wg.Wait()
	}
}
