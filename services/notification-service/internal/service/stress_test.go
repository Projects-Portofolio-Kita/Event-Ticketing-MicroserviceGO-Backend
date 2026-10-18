package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
)

// TestNotificationService_Concurrent1000Submissions tests submitting 1000 notification jobs
// concurrently across 50 goroutines to ensure the service layer handles high concurrent volume.
func TestNotificationService_Concurrent1000Submissions(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to create mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "noreply@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create smtp client: %v", err)
	}

	// Worker pool with high capacity for 1000 submissions
	pool := queue.NewWorkerPool(8, 1500)
	poolCtx, cancelPool := context.WithCancel(context.Background())
	defer cancelPool()
	pool.Start(poolCtx)

	svc, err := service.NewNotificationService(client, pool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialize notification service: %v", err)
	}

	const totalRequests = 1000
	const concurrentSenders = 50
	const requestsPerSender = totalRequests / concurrentSenders

	var acceptedCount atomic.Int32
	var submitErrors atomic.Int32
	var wg sync.WaitGroup

	startTime := time.Now()

	for s := 0; s < concurrentSenders; s++ {
		wg.Add(1)
		go func(senderID int) {
			defer wg.Done()
			for i := 0; i < requestsPerSender; i++ {
				req := dto.CustomEmailRequest{
					To:      []string{fmt.Sprintf("user_%d_%d@example.com", senderID, i)},
					Subject: fmt.Sprintf("High Volume Notification %d-%d", senderID, i),
					Body:    "Your transaction is confirmed.",
					IsHTML:  false,
				}

				jobID, err := svc.SendCustomEmail(context.Background(), req)
				if err != nil {
					submitErrors.Add(1)
				} else if jobID != "" {
					acceptedCount.Add(1)
				}
			}
		}(s)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("Submitted %d jobs concurrently in %v (Accepted: %d, Errors: %d)",
		totalRequests, elapsed, acceptedCount.Load(), submitErrors.Load())

	if submitErrors.Load() > 0 {
		t.Errorf("encountered %d submission errors under 1000 job load", submitErrors.Load())
	}
	if acceptedCount.Load() != totalRequests {
		t.Errorf("expected %d accepted jobs, got %d", totalRequests, acceptedCount.Load())
	}

	// Stop pool gracefully with timeout
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	_ = pool.Stop(stopCtx)
}

// TestNotificationService_SaturationRejection verifies that when the worker pool queue is saturated,
// the service layer immediately returns ErrQueueFull.
func TestNotificationService_SaturationRejection(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to create mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "noreply@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create smtp client: %v", err)
	}

	// Pool with tiny capacity of 5, workers NOT started so channel saturates
	pool := queue.NewWorkerPool(1, 5)

	svc, err := service.NewNotificationService(client, pool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialize notification service: %v", err)
	}

	// Fill the 5 slots
	for i := 0; i < 5; i++ {
		req := dto.CustomEmailRequest{
			To:      []string{fmt.Sprintf("fill%d@example.com", i)},
			Subject: "Buffer fill",
			Body:    "Filling buffer",
		}
		_, err := svc.SendCustomEmail(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to fill buffer slot %d: %v", i, err)
		}
	}

	// 6th request MUST fail with queue.ErrQueueFull
	req := dto.CustomEmailRequest{
		To:      []string{"overflow@example.com"},
		Subject: "Overflow",
		Body:    "This should overflow",
	}
	_, err = svc.SendCustomEmail(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for saturated queue, got nil")
	}

	if !errors.Is(err, queue.ErrQueueFull) {
		t.Fatalf("expected errors.Is(err, queue.ErrQueueFull) to be true, got: %v", err)
	}
}

// TestNotificationService_FullEndToEndTicketConfirmationStress tests concurrent submission
// of full TicketConfirmationRequests (involving QR code generation and PDF building).
func TestNotificationService_FullEndToEndTicketConfirmationStress(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to create mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "noreply@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create smtp client: %v", err)
	}

	// 4 workers, 100 capacity
	pool := queue.NewWorkerPool(4, 100)
	ctx := context.Background()
	pool.Start(ctx)

	svc, err := service.NewNotificationService(client, pool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialize notification service: %v", err)
	}

	const ticketRequests = 20
	var wg sync.WaitGroup

	for i := 0; i < ticketRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := dto.TicketConfirmationRequest{
				BookingRef:    fmt.Sprintf("TICK-STRESS-%04d", idx),
				CustomerEmail: fmt.Sprintf("attendee%d@example.com", idx),
				CustomerName:  fmt.Sprintf("Attendee %d", idx),
				EventID:       "evt-stress-999",
				EventTitle:    "Coldplay Music of the Spheres",
				EventDate:     "Sunday, 15 Nov 2026, 20:00 WIB",
				VenueName:     "Gelora Bung Karno",
				VenueAddress:  "Senayan, Jakarta",
				TierName:      "Cat 1",
				TotalAmount:   5000000,
				Currency:      "IDR",
				Seats: []dto.SeatInfoDTO{
					{
						SeatID:  fmt.Sprintf("seat-%d-1", idx),
						Section: "Cat 1A",
						Row:     "10",
						Number:  fmt.Sprintf("10%02d", idx),
						Price:   2500000,
					},
					{
						SeatID:  fmt.Sprintf("seat-%d-2", idx),
						Section: "Cat 1A",
						Row:     "10",
						Number:  fmt.Sprintf("10%02d", idx+1),
						Price:   2500000,
					},
				},
			}

			_, err := svc.SendTicketConfirmation(context.Background(), req)
			if err != nil {
				t.Errorf("failed to submit ticket confirmation %d: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// Drain the worker pool
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	if err := pool.Stop(stopCtx); err != nil {
		t.Fatalf("failed to gracefully drain pool: %v", err)
	}

	// Verify all 20 emails were delivered to mock SMTP server with attachments
	if mockServer.MessageCount() != ticketRequests {
		t.Errorf("expected %d emails received by mock server, got %d", ticketRequests, mockServer.MessageCount())
	}

	messages := mockServer.GetReceivedMessages()
	for _, msg := range messages {
		if len(msg.Attachments) == 0 {
			t.Errorf("expected PDF attachment on ticket confirmation email to %v, got none", msg.To)
		}
	}
}
