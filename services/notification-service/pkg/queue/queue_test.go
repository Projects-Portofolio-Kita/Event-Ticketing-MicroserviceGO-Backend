package queue_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
)

func TestWorkerPool_DefaultsAndAttributes(t *testing.T) {
	wp := queue.NewWorkerPool(0, 0)
	if wp.WorkerCount() != 5 {
		t.Errorf("expected default worker count 5, got %d", wp.WorkerCount())
	}
	if wp.Capacity() != 500 {
		t.Errorf("expected default capacity 500, got %d", wp.Capacity())
	}
	if wp.PendingJobs() != 0 {
		t.Errorf("expected 0 pending jobs, got %d", wp.PendingJobs())
	}
}

func TestWorkerPool_ConcurrentExecution(t *testing.T) {
	wp := queue.NewWorkerPool(4, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	const totalJobs = 50
	var executedCount atomic.Int32
	var wg sync.WaitGroup
	wg.Add(totalJobs)

	for i := 0; i < totalJobs; i++ {
		jobID := time.Now().Format("150405.000000")
		job := queue.NewJob(jobID, func(ctx context.Context) error {
			defer wg.Done()
			executedCount.Add(1)
			time.Sleep(2 * time.Millisecond) // simulate brief work
			return nil
		})

		if err := wp.Submit(job); err != nil {
			t.Fatalf("failed to submit job %d: %v", i, err)
		}
	}

	wg.Wait()

	if executedCount.Load() != totalJobs {
		t.Errorf("expected %d executed jobs, got %d", totalJobs, executedCount.Load())
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("unexpected error stopping pool: %v", err)
	}
}

func TestWorkerPool_GracefulDraining(t *testing.T) {
	// Create pool with 1 worker and capacity 20
	wp := queue.NewWorkerPool(1, 20)
	ctx := context.Background()

	// Fill queue BEFORE starting workers
	const jobCount = 10
	var executedCount atomic.Int32

	for i := 0; i < jobCount; i++ {
		job := queue.NewJob("drain-job", func(ctx context.Context) error {
			executedCount.Add(1)
			time.Sleep(5 * time.Millisecond)
			return nil
		})
		if err := wp.Submit(job); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	if wp.PendingJobs() != jobCount {
		t.Errorf("expected %d pending jobs, got %d", jobCount, wp.PendingJobs())
	}

	// Start workers and immediately initiate Stop
	wp.Start(ctx)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	// Stop must wait for all buffered jobs to finish draining
	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	if executedCount.Load() != jobCount {
		t.Errorf("expected all %d jobs drained, but only %d executed", jobCount, executedCount.Load())
	}
	if wp.PendingJobs() != 0 {
		t.Errorf("expected 0 pending jobs after drain, got %d", wp.PendingJobs())
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	// 1 worker, small capacity of 2
	wp := queue.NewWorkerPool(1, 2)
	// Do NOT start workers yet so channel fills up

	job1 := queue.NewJob("job1", func(ctx context.Context) error { return nil })
	job2 := queue.NewJob("job2", func(ctx context.Context) error { return nil })
	job3 := queue.NewJob("job3", func(ctx context.Context) error { return nil })

	if err := wp.Submit(job1); err != nil {
		t.Fatalf("unexpected error for job1: %v", err)
	}
	if err := wp.Submit(job2); err != nil {
		t.Fatalf("unexpected error for job2: %v", err)
	}

	// 3rd job should overflow capacity
	err := wp.Submit(job3)
	if !errors.Is(err, queue.ErrQueueFull) {
		t.Errorf("expected ErrQueueFull, got: %v", err)
	}
}

func TestWorkerPool_SubmitAfterStop(t *testing.T) {
	wp := queue.NewWorkerPool(2, 10)
	wp.Start(context.Background())

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	job := queue.NewJob("post-stop", func(ctx context.Context) error { return nil })
	err := wp.Submit(job)
	if !errors.Is(err, queue.ErrQueueShutdown) {
		t.Errorf("expected ErrQueueShutdown, got: %v", err)
	}
}

func TestWorkerPool_NilJob(t *testing.T) {
	wp := queue.NewWorkerPool(2, 10)
	err := wp.Submit(nil)
	if !errors.Is(err, queue.ErrJobNil) {
		t.Errorf("expected ErrJobNil, got: %v", err)
	}
}

func TestWorkerPool_ShutdownTimeout(t *testing.T) {
	wp := queue.NewWorkerPool(1, 5)
	wp.Start(context.Background())

	// Job that blocks for longer than shutdown timeout
	slowJob := queue.NewJob("slow", func(ctx context.Context) error {
		time.Sleep(300 * time.Millisecond)
		return nil
	})
	_ = wp.Submit(slowJob)

	// Stop with very tight timeout of 10ms
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stopCancel()

	err := wp.Stop(stopCtx)
	if err == nil {
		t.Fatal("expected timeout error on Stop, got nil")
	}
}

func TestWorkerPool_JobPanicRecovery(t *testing.T) {
	// Single worker to ensure that if this worker dies, no other worker exists to pick up the second job
	wp := queue.NewWorkerPool(1, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	var secondJobExecuted atomic.Bool
	var wg sync.WaitGroup
	wg.Add(2)

	// Job 1 panics deliberately
	panicJob := queue.NewJob("panic-job", func(ctx context.Context) error {
		defer wg.Done()
		panic("deliberate test panic in job")
	})

	// Job 2 must still execute on the same worker
	normalJob := queue.NewJob("normal-job", func(ctx context.Context) error {
		defer wg.Done()
		secondJobExecuted.Store(true)
		return nil
	})

	if err := wp.Submit(panicJob); err != nil {
		t.Fatalf("failed to submit panicJob: %v", err)
	}
	if err := wp.Submit(normalJob); err != nil {
		t.Fatalf("failed to submit normalJob: %v", err)
	}

	wg.Wait()

	if !secondJobExecuted.Load() {
		t.Fatal("expected second job to execute after panic in previous job, but worker died")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := wp.Stop(stopCtx); err != nil {
		t.Fatalf("unexpected error stopping pool: %v", err)
	}
}
