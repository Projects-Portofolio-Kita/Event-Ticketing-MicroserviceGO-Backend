package smtp_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
)

// TestMockSMTPServer_HighConcurrency tests multiple concurrent workers
// connecting and sending emails to MockSMTPServer using the same smtp.Client.
func TestMockSMTPServer_HighConcurrency(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to create mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "sender@example.com", false)
	if err != nil {
		t.Fatalf("failed to create smtp client: %v", err)
	}

	const numWorkers = 20
	const msgsPerWorker = 25
	const totalMsgs = numWorkers * msgsPerWorker

	var successCount atomic.Int32
	var errorCount atomic.Int32
	var wg sync.WaitGroup

	startTime := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < msgsPerWorker; i++ {
				msg := &smtp.EmailMessage{
					To:       []string{fmt.Sprintf("worker%d_msg%d@example.com", workerID, i)},
					Subject:  fmt.Sprintf("Concurrent Subject %d-%d", workerID, i),
					TextBody: "Hello concurrent world",
					HTMLBody: "<p>Hello concurrent world</p>",
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := client.Send(ctx, msg)
				cancel()

				if err != nil {
					errorCount.Add(1)
					t.Logf("Worker %d msg %d send error: %v", workerID, i, err)
				} else {
					successCount.Add(1)
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(startTime)
	t.Logf("Finished %d sends in %v (Success: %d, Errors: %d)", totalMsgs, duration, successCount.Load(), errorCount.Load())

	if errorCount.Load() > 0 {
		t.Errorf("encountered %d send errors during concurrent execution", errorCount.Load())
	}

	receivedCount := mockServer.MessageCount()
	if receivedCount != totalMsgs {
		t.Errorf("expected %d messages in mock server, got %d", totalMsgs, receivedCount)
	}
}

// TestMockSMTPServer_ConcurrentAttachments tests multiple workers concurrently sending emails
// with binary attachments to ensure multipart body handling and memory buffers are race-free.
func TestMockSMTPServer_ConcurrentAttachments(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to create mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "sender@example.com", false)
	if err != nil {
		t.Fatalf("failed to create smtp client: %v", err)
	}

	const numWorkers = 10
	const msgsPerWorker = 20
	const totalMsgs = numWorkers * msgsPerWorker

	var successCount atomic.Int32
	var wg sync.WaitGroup

	dummyPDF := []byte("%PDF-1.4 mock binary pdf data with various bytes \x00\x01\x02\xff\xfe")

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < msgsPerWorker; i++ {
				msg := &smtp.EmailMessage{
					To:       []string{fmt.Sprintf("att_worker%d_%d@example.com", workerID, i)},
					Subject:  fmt.Sprintf("Attachment Test %d-%d", workerID, i),
					TextBody: "Please find attached ticket.",
					HTMLBody: "<p>Please find attached ticket.</p>",
					Attachments: []smtp.Attachment{
						{
							Filename:    fmt.Sprintf("ticket_%d_%d.pdf", workerID, i),
							ContentType: "application/pdf",
							Data:        dummyPDF,
						},
					},
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := client.Send(ctx, msg)
				cancel()

				if err != nil {
					t.Errorf("Worker %d msg %d send error: %v", workerID, i, err)
				} else {
					successCount.Add(1)
				}
			}
		}(w)
	}

	wg.Wait()

	if successCount.Load() != totalMsgs {
		t.Errorf("expected %d successes, got %d", totalMsgs, successCount.Load())
	}
	if mockServer.MessageCount() != totalMsgs {
		t.Errorf("expected %d received messages, got %d", totalMsgs, mockServer.MessageCount())
	}

	// Verify attachments captured accurately
	messages := mockServer.GetReceivedMessages()
	for _, m := range messages {
		if len(m.Attachments) != 1 {
			t.Fatalf("expected 1 attachment per message, got %d", len(m.Attachments))
		}
		if m.Attachments[0].ContentType != "application/pdf" {
			t.Errorf("expected application/pdf, got %s", m.Attachments[0].ContentType)
		}
		if len(m.Attachments[0].Data) != len(dummyPDF) {
			t.Errorf("expected %d bytes, got %d", len(dummyPDF), len(m.Attachments[0].Data))
		}
	}
}

