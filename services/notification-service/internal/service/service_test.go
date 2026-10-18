package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	"github.com/google/uuid"
)

// setupTestEnvironment spins up an in-memory mock SMTP server, SMTP client, and worker pool.
func setupTestEnvironment(t *testing.T, workerCount, queueCapacity int) (*smtp.MockSMTPServer, queue.WorkerPool, NotificationService) {
	t.Helper()

	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}

	client, err := smtp.NewClient(
		mockServer.Host(),
		mockServer.Port(),
		"", "",
		"noreply@tickid.com",
		false,
	)
	if err != nil {
		mockServer.Close()
		t.Fatalf("failed to create smtp client: %v", err)
	}

	pool := queue.NewWorkerPool(workerCount, queueCapacity)
	ctx := context.Background()
	pool.Start(ctx)

	svc, err := NewNotificationService(client, pool, nil, nil, nil)
	if err != nil {
		pool.Stop(ctx)
		mockServer.Close()
		t.Fatalf("failed to initialize notification service: %v", err)
	}

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = pool.Stop(stopCtx)
		_ = mockServer.Close()
	})

	return mockServer, pool, svc
}

// waitForMessages polls the mock server until expectedCount messages are received or timeout expires.
func waitForMessages(t *testing.T, server *smtp.MockSMTPServer, expectedCount int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if server.MessageCount() >= expectedCount {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestSendTicketConfirmation_Success(t *testing.T) {
	mockServer, _, svc := setupTestEnvironment(t, 2, 10)
	ctx := context.Background()

	req := dto.TicketConfirmationRequest{
		BookingRef:    "TICK-TEST-1001",
		CustomerEmail: "attendee@example.com",
		CustomerName:  "Jane Attendee",
		EventID:       "evt-2026-xyz",
		EventTitle:    "Bruno Mars Live in Jakarta",
		EventDate:     "Saturday, 12 December 2026, 19:30 WIB",
		VenueName:     "Jakarta International Stadium",
		VenueAddress:  "Tanjung Priok, Jakarta Utara",
		TierName:      "Diamond VIP",
		TotalAmount:   7500000,
		Currency:      "IDR",
		Seats: []dto.SeatInfoDTO{
			{SeatID: "s-101", Section: "Diamond A", Row: "1", Number: "A01", Price: 3750000},
			{SeatID: "s-102", Section: "Diamond A", Row: "1", Number: "A02", Price: 3750000},
		},
	}

	jobID, err := svc.SendTicketConfirmation(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error submitting ticket confirmation: %v", err)
	}

	// 1. Verify JobID format
	if _, err := uuid.Parse(jobID); err != nil {
		t.Fatalf("expected valid UUID for jobID, got: %s (err: %v)", jobID, err)
	}

	// 2. Wait for asynchronous worker pool to process and deliver to mock SMTP
	if !waitForMessages(t, mockServer, 1, 4*time.Second) {
		t.Fatalf("timed out waiting for ticket confirmation email to arrive at mock SMTP server (count=%d)", mockServer.MessageCount())
	}

	msg := mockServer.LastMessage()
	if msg == nil {
		t.Fatal("expected received message to be non-nil")
	}

	// 3. Assert Recipient and Subject
	if len(msg.To) != 1 || msg.To[0] != "attendee@example.com" {
		t.Errorf("expected recipient attendee@example.com, got: %v", msg.To)
	}
	if !strings.Contains(msg.Subject, "Bruno Mars Live in Jakarta") || !strings.Contains(msg.Subject, "TICK-TEST-1001") {
		t.Errorf("subject missing event title or booking reference: %s", msg.Subject)
	}

	// 4. Assert HTML and Text Content
	if !strings.Contains(msg.HTMLBody, "Jane Attendee") {
		t.Errorf("HTML body missing customer name")
	}
	if !strings.Contains(msg.HTMLBody, "TICK-TEST-1001") {
		t.Errorf("HTML body missing booking reference")
	}
	if !strings.Contains(msg.HTMLBody, "Diamond VIP") {
		t.Errorf("HTML body missing tier name")
	}
	if !strings.Contains(msg.HTMLBody, "Jakarta International Stadium") {
		t.Errorf("HTML body missing venue name")
	}
	if !strings.Contains(msg.HTMLBody, "ticket_TICK-TEST-1001.pdf") {
		t.Errorf("HTML body missing PDF attachment notice")
	}
	if !strings.Contains(msg.HTMLBody, "data:image/png;base64,") {
		t.Errorf("HTML body missing base64 QR code")
	}

	if !strings.Contains(msg.TextBody, "TICK-TEST-1001") {
		t.Errorf("Text body missing booking reference")
	}

	// 5. Assert PDF Attachment
	if len(msg.Attachments) != 1 {
		t.Fatalf("expected exactly 1 attachment, got %d", len(msg.Attachments))
	}

	att := msg.Attachments[0]
	if att.Filename != "ticket_TICK-TEST-1001.pdf" {
		t.Errorf("expected attachment filename ticket_TICK-TEST-1001.pdf, got: %s", att.Filename)
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("expected ContentType application/pdf, got: %s", att.ContentType)
	}
	if !bytes.HasPrefix(att.Data, []byte("%PDF-")) {
		t.Errorf("expected attachment data to start with %%PDF- magic header")
	}
	if len(att.Data) < 10000 {
		t.Errorf("expected generated PDF size > 10KB, got %d bytes", len(att.Data))
	}
}

func TestSendPaymentFailed_Success(t *testing.T) {
	mockServer, _, svc := setupTestEnvironment(t, 2, 10)
	ctx := context.Background()

	req := dto.PaymentFailedRequest{
		BookingRef:    "TICK-FAIL-9002",
		CustomerEmail: "unpaid@example.com",
		CustomerName:  "Bob Builder",
		EventTitle:    "Java Rockin Land 2026",
		Reason:        "Payment gateway bank transfer expired after 15 minutes",
		ExpiredAt:     "2026-10-18 22:00:00 WIB",
	}

	jobID, err := svc.SendPaymentFailed(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error submitting payment failed: %v", err)
	}

	if _, err := uuid.Parse(jobID); err != nil {
		t.Fatalf("expected valid UUID for jobID, got: %s", jobID)
	}

	if !waitForMessages(t, mockServer, 1, 3*time.Second) {
		t.Fatalf("timed out waiting for payment failed email")
	}

	msg := mockServer.LastMessage()
	if msg == nil {
		t.Fatal("expected message to be non-nil")
	}

	if len(msg.To) != 1 || msg.To[0] != "unpaid@example.com" {
		t.Errorf("expected recipient unpaid@example.com, got: %v", msg.To)
	}
	if !strings.Contains(msg.Subject, "Payment Expired") || !strings.Contains(msg.Subject, "TICK-FAIL-9002") {
		t.Errorf("subject missing expected details: %s", msg.Subject)
	}
	if !strings.Contains(msg.HTMLBody, "Bob Builder") {
		t.Errorf("HTML missing customer name")
	}
	if !strings.Contains(msg.HTMLBody, "Payment gateway bank transfer expired") {
		t.Errorf("HTML missing failure reason")
	}
	if !strings.Contains(msg.HTMLBody, "Seats Released") {
		t.Errorf("HTML missing released notice")
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("expected 0 attachments for payment failed, got %d", len(msg.Attachments))
	}
}

func TestSendCustomEmail_Success(t *testing.T) {
	mockServer, _, svc := setupTestEnvironment(t, 2, 10)
	ctx := context.Background()

	// HTML custom email
	reqHTML := dto.CustomEmailRequest{
		To:      []string{"member1@example.com", "member2@example.com"},
		Subject: "TickId Early Bird Presale Announcement",
		Body:    "<h1>Exclusive Presale Access</h1><p>Use code PRESALE2026</p>",
		IsHTML:  true,
	}

	jobID, err := svc.SendCustomEmail(ctx, reqHTML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := uuid.Parse(jobID); err != nil {
		t.Fatalf("expected valid UUID, got: %s", jobID)
	}

	if !waitForMessages(t, mockServer, 1, 3*time.Second) {
		t.Fatalf("timed out waiting for custom email")
	}

	msg := mockServer.LastMessage()
	if len(msg.To) != 2 {
		t.Errorf("expected 2 recipients, got: %v", msg.To)
	}
	if msg.Subject != "TickId Early Bird Presale Announcement" {
		t.Errorf("unexpected subject: %s", msg.Subject)
	}
	if !strings.Contains(msg.HTMLBody, "<h1>Exclusive Presale Access</h1>") {
		t.Errorf("HTML body missing content: %s", msg.HTMLBody)
	}
}

func TestInputValidation_TicketConfirmation(t *testing.T) {
	_, _, svc := setupTestEnvironment(t, 1, 5)
	ctx := context.Background()

	validReq := dto.TicketConfirmationRequest{
		BookingRef:    "TICK-VAL-001",
		CustomerEmail: "valid@example.com",
		CustomerName:  "Valid Name",
		EventID:       "e-1",
		EventTitle:    "Valid Event",
		EventDate:     "Tomorrow",
		VenueName:     "Arena",
		TierName:      "Gold",
		TotalAmount:   100000,
		Seats: []dto.SeatInfoDTO{
			{SeatID: "s-1", Number: "01"},
		},
	}

	tests := []struct {
		name        string
		modify      func(r *dto.TicketConfirmationRequest)
		expectedErr error
	}{
		{
			name:        "empty booking ref",
			modify:      func(r *dto.TicketConfirmationRequest) { r.BookingRef = "   " },
			expectedErr: ErrEmptyBookingRef,
		},
		{
			name:        "empty customer email",
			modify:      func(r *dto.TicketConfirmationRequest) { r.CustomerEmail = "" },
			expectedErr: ErrInvalidEmail,
		},
		{
			name:        "malformed email without @",
			modify:      func(r *dto.TicketConfirmationRequest) { r.CustomerEmail = "invalid-email" },
			expectedErr: ErrInvalidEmail,
		},
		{
			name:        "malformed email without domain dot",
			modify:      func(r *dto.TicketConfirmationRequest) { r.CustomerEmail = "user@domain" },
			expectedErr: ErrInvalidEmail,
		},
		{
			name:        "empty customer name",
			modify:      func(r *dto.TicketConfirmationRequest) { r.CustomerName = "" },
			expectedErr: ErrEmptyCustomerName,
		},
		{
			name:        "empty event title",
			modify:      func(r *dto.TicketConfirmationRequest) { r.EventTitle = "" },
			expectedErr: ErrEmptyEventTitle,
		},
		{
			name:        "empty seats slice",
			modify:      func(r *dto.TicketConfirmationRequest) { r.Seats = nil },
			expectedErr: ErrEmptySeats,
		},
		{
			name: "seat with missing seat_id",
			modify: func(r *dto.TicketConfirmationRequest) {
				r.Seats = []dto.SeatInfoDTO{{SeatID: "", Number: "01"}}
			},
			expectedErr: errors.New("seat_id and number are required for every seat"),
		},
		{
			name: "seat with missing number",
			modify: func(r *dto.TicketConfirmationRequest) {
				r.Seats = []dto.SeatInfoDTO{{SeatID: "s-1", Number: ""}}
			},
			expectedErr: errors.New("seat_id and number are required for every seat"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqCopy := validReq
			tc.modify(&reqCopy)
			_, err := svc.SendTicketConfirmation(ctx, reqCopy)
			if err == nil {
				t.Fatalf("expected error for case %s, got nil", tc.name)
			}
			if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) && !strings.Contains(err.Error(), tc.expectedErr.Error()) {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestInputValidation_PaymentFailed(t *testing.T) {
	_, _, svc := setupTestEnvironment(t, 1, 5)
	ctx := context.Background()

	validReq := dto.PaymentFailedRequest{
		BookingRef:    "TICK-FAIL-VAL",
		CustomerEmail: "val@example.com",
		CustomerName:  "Test User",
		EventTitle:    "Test Festival",
		Reason:        "Timeout",
	}

	tests := []struct {
		name        string
		modify      func(r *dto.PaymentFailedRequest)
		expectedErr error
	}{
		{
			name:        "empty booking ref",
			modify:      func(r *dto.PaymentFailedRequest) { r.BookingRef = "" },
			expectedErr: ErrEmptyBookingRef,
		},
		{
			name:        "invalid email",
			modify:      func(r *dto.PaymentFailedRequest) { r.CustomerEmail = "bad-email@" },
			expectedErr: ErrInvalidEmail,
		},
		{
			name:        "empty customer name",
			modify:      func(r *dto.PaymentFailedRequest) { r.CustomerName = "" },
			expectedErr: ErrEmptyCustomerName,
		},
		{
			name:        "empty event title",
			modify:      func(r *dto.PaymentFailedRequest) { r.EventTitle = "" },
			expectedErr: ErrEmptyEventTitle,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqCopy := validReq
			tc.modify(&reqCopy)
			_, err := svc.SendPaymentFailed(ctx, reqCopy)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestInputValidation_CustomEmail(t *testing.T) {
	_, _, svc := setupTestEnvironment(t, 1, 5)
	ctx := context.Background()

	tests := []struct {
		name        string
		req         dto.CustomEmailRequest
		expectedErr error
	}{
		{
			name: "empty to recipients",
			req: dto.CustomEmailRequest{
				To:      nil,
				Subject: "Sub",
				Body:    "Body",
			},
			expectedErr: ErrEmptyRecipients,
		},
		{
			name: "invalid recipient email",
			req: dto.CustomEmailRequest{
				To:      []string{"valid@example.com", "not-an-email"},
				Subject: "Sub",
				Body:    "Body",
			},
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "empty subject",
			req: dto.CustomEmailRequest{
				To:      []string{"valid@example.com"},
				Subject: "",
				Body:    "Body",
			},
			expectedErr: ErrEmptySubject,
		},
		{
			name: "empty body",
			req: dto.CustomEmailRequest{
				To:      []string{"valid@example.com"},
				Subject: "Sub",
				Body:    "",
			},
			expectedErr: ErrEmptyBody,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SendCustomEmail(ctx, tc.req)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) && !strings.Contains(err.Error(), tc.expectedErr.Error()) {
				t.Errorf("expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestWorkerPool_SaturationHandling(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "test@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Pool with capacity 1 and 0 workers running so jobs queue up and saturate capacity
	pool := queue.NewWorkerPool(1, 1)
	// Notice: we do NOT call pool.Start, so the single slot remains occupied
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = pool.Stop(stopCtx)
	}()

	svc, err := NewNotificationService(client, pool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	req := dto.CustomEmailRequest{
		To:      []string{"target@example.com"},
		Subject: "Saturation Test 1",
		Body:    "Body 1",
	}

	// First submit occupies capacity 1
	job1, err := svc.SendCustomEmail(ctx, req)
	if err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	if job1 == "" {
		t.Fatal("expected non-empty jobID")
	}

	// Second submit should encounter ErrQueueFull
	_, err = svc.SendCustomEmail(ctx, req)
	if err == nil {
		t.Fatal("expected queue saturation error, got nil")
	}
	if !strings.Contains(err.Error(), "queue is full") && !errors.Is(err, queue.ErrQueueFull) {
		t.Errorf("expected queue full error, got: %v", err)
	}
}

func TestNewNotificationService_NilChecks(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "test@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	pool := queue.NewWorkerPool(1, 5)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = pool.Stop(stopCtx)
	}()

	// 1. Nil EmailSender
	_, err = NewNotificationService(nil, pool, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "emailSender cannot be nil") {
		t.Errorf("expected error for nil emailSender, got: %v", err)
	}

	// 2. Nil WorkerPool
	_, err = NewNotificationService(client, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "workerPool cannot be nil") {
		t.Errorf("expected error for nil workerPool, got: %v", err)
	}

	// 3. Valid with nil optional deps
	svc, err := NewNotificationService(client, pool, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected successful instantiation with default sub-components: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}
