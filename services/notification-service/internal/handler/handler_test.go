package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/pdf"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestEnvironment(t *testing.T) (*gin.Engine, *smtp.MockSMTPServer, *queue.Pool) {
	t.Helper()

	mockSMTP, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}

	smtpClient, err := smtp.NewClient(
		"127.0.0.1",
		mockSMTP.Port(),
		"",
		"",
		"noreply@tickid.com",
		false,
	)
	if err != nil {
		mockSMTP.Close()
		t.Fatalf("failed to create smtp client: %v", err)
	}

	workerPool := queue.NewWorkerPool(2, 50)
	workerPool.Start(context.Background())

	qrGen := pdf.NewQRCodeGenerator(pdf.DefaultQRSize)
	pdfGen := pdf.NewTicketPDFGenerator(qrGen)
	tmplEngine, err := template.NewEngine()
	if err != nil {
		mockSMTP.Close()
		t.Fatalf("failed to create template engine: %v", err)
	}

	notifService, err := service.NewNotificationService(smtpClient, workerPool, pdfGen, qrGen, tmplEngine)
	if err != nil {
		mockSMTP.Close()
		t.Fatalf("failed to create notification service: %v", err)
	}

	notifHandler := handler.NewNotificationHandler(notifService)
	healthHandler := handler.NewHealthHandler()
	router := handler.SetupRouter(notifHandler, healthHandler)

	return router, mockSMTP, workerPool
}

func TestHealthCheck(t *testing.T) {
	healthHandler := handler.NewHealthHandler()
	router := handler.SetupRouter(nil, healthHandler)

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res handler.HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", res.Status)
	}
	if res.Service != "notification-service" {
		t.Errorf("expected service 'notification-service', got %q", res.Service)
	}
}

func TestTicketConfirmation_Success(t *testing.T) {
	router, mockSMTP, workerPool := setupTestEnvironment(t)
	defer mockSMTP.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = workerPool.Stop(ctx)
	}()

	payload := dto.TicketConfirmationRequest{
		BookingRef:    "TICK-TEST-1001",
		CustomerEmail: "customer@example.com",
		CustomerName:  "Alex Johnson",
		EventID:       "EVT-7788",
		EventTitle:    "Rock Legends World Tour 2026",
		EventDate:     "Saturday, Nov 21, 2026 at 19:30 WIB",
		VenueName:     "Gelora Bung Karno Main Stadium",
		VenueAddress:  "Jl. Pintu Satu Senayan, Gelora, Jakarta",
		TierName:      "VIP Platinum",
		TotalAmount:   2500000,
		Currency:      "IDR",
		Seats: []dto.SeatInfoDTO{
			{
				SeatID:  "SEAT-001",
				Section: "Platinum-A",
				Row:     "R-01",
				Number:  "A-12",
				Price:   2500000,
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/notifications/ticket-confirmation", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 Accepted, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res dto.BaseResponse[dto.QueuedResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	if res.Message != "Ticket confirmation notification queued for delivery" {
		t.Errorf("unexpected message: %s", res.Message)
	}
	if res.Data.JobID == "" {
		t.Errorf("expected non-empty job_id in data")
	}
	if _, err := uuid.Parse(res.Data.JobID); err != nil {
		t.Errorf("expected valid UUID for job_id, got %s: %v", res.Data.JobID, err)
	}
}

func TestTicketConfirmation_ValidationFailures(t *testing.T) {
	router, mockSMTP, workerPool := setupTestEnvironment(t)
	defer mockSMTP.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = workerPool.Stop(ctx)
	}()

	tests := []struct {
		name        string
		body        string
		description string
	}{
		{
			name:        "malformed_json",
			body:        `{"booking_ref": "TICK-1002", invalid json`,
			description: "Malformed JSON body syntax",
		},
		{
			name: "missing_booking_ref",
			body: `{
				"customer_email": "customer@example.com",
				"customer_name": "Alex Johnson",
				"event_id": "EVT-1",
				"event_title": "Concert",
				"event_date": "2026-11-20",
				"venue_name": "Stadium",
				"tier_name": "VIP",
				"total_amount": 100000,
				"seats": [{"seat_id": "S1", "number": "1"}]
			}`,
			description: "Missing required booking_ref",
		},
		{
			name: "invalid_email_format",
			body: `{
				"booking_ref": "TICK-1003",
				"customer_email": "not-an-email",
				"customer_name": "Alex Johnson",
				"event_id": "EVT-1",
				"event_title": "Concert",
				"event_date": "2026-11-20",
				"venue_name": "Stadium",
				"tier_name": "VIP",
				"total_amount": 100000,
				"seats": [{"seat_id": "S1", "number": "1"}]
			}`,
			description: "Customer email missing @ or domain",
		},
		{
			name: "empty_seats_slice",
			body: `{
				"booking_ref": "TICK-1004",
				"customer_email": "customer@example.com",
				"customer_name": "Alex Johnson",
				"event_id": "EVT-1",
				"event_title": "Concert",
				"event_date": "2026-11-20",
				"venue_name": "Stadium",
				"tier_name": "VIP",
				"total_amount": 100000,
				"seats": []
			}`,
			description: "Seats slice is empty",
		},
		{
			name: "missing_seat_number",
			body: `{
				"booking_ref": "TICK-1005",
				"customer_email": "customer@example.com",
				"customer_name": "Alex Johnson",
				"event_id": "EVT-1",
				"event_title": "Concert",
				"event_date": "2026-11-20",
				"venue_name": "Stadium",
				"tier_name": "VIP",
				"total_amount": 100000,
				"seats": [{"seat_id": "S1"}]
			}`,
			description: "Seat missing required number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/api/v1/notifications/ticket-confirmation", bytes.NewBufferString(tt.body))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("[%s] expected 400 Bad Request, got %d. Body: %s", tt.name, w.Code, w.Body.String())
			}

			var res dto.BaseResponse[any]
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatalf("[%s] failed to decode error response: %v", tt.name, err)
			}

			if res.Success {
				t.Errorf("[%s] expected success false, got true", tt.name)
			}
			if res.Message != "Validation failed" {
				t.Errorf("[%s] expected Message 'Validation failed', got %q", tt.name, res.Message)
			}
			if res.Error == "" {
				t.Errorf("[%s] expected non-empty error field", tt.name)
			}
		})
	}
}

func TestPaymentFailed_Success(t *testing.T) {
	router, mockSMTP, workerPool := setupTestEnvironment(t)
	defer mockSMTP.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = workerPool.Stop(ctx)
	}()

	payload := dto.PaymentFailedRequest{
		BookingRef:    "TICK-FAIL-2001",
		CustomerEmail: "unpaid@example.com",
		CustomerName:  "Jane Smith",
		EventTitle:    "Jakarta International Jazz Festival",
		Reason:        "Payment time limit exceeded (15 minutes expired)",
		ExpiredAt:     "2026-10-18 22:30 WIB",
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/notifications/payment-failed", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 Accepted, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res dto.BaseResponse[dto.QueuedResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	if res.Message != "Payment failed notification queued for delivery" {
		t.Errorf("unexpected message: %s", res.Message)
	}
	if res.Data.JobID == "" {
		t.Errorf("expected non-empty job_id in data")
	}
	if _, err := uuid.Parse(res.Data.JobID); err != nil {
		t.Errorf("expected valid UUID for job_id, got %s: %v", res.Data.JobID, err)
	}
}

func TestPaymentFailed_ValidationFailures(t *testing.T) {
	router, mockSMTP, workerPool := setupTestEnvironment(t)
	defer mockSMTP.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = workerPool.Stop(ctx)
	}()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed_json",
			body: `{"booking_ref": "TICK-2002", not valid`,
		},
		{
			name: "missing_booking_ref",
			body: `{
				"customer_email": "user@example.com",
				"customer_name": "Jane Smith",
				"event_title": "Jazz Fest",
				"reason": "Payment window expired"
			}`,
		},
		{
			name: "invalid_email",
			body: `{
				"booking_ref": "TICK-2003",
				"customer_email": "invalid-email-address",
				"customer_name": "Jane Smith",
				"event_title": "Jazz Fest",
				"reason": "Payment window expired"
			}`,
		},
		{
			name: "missing_customer_name",
			body: `{
				"booking_ref": "TICK-2004",
				"customer_email": "user@example.com",
				"event_title": "Jazz Fest",
				"reason": "Payment window expired"
			}`,
		},
		{
			name: "missing_reason",
			body: `{
				"booking_ref": "TICK-2005",
				"customer_email": "user@example.com",
				"customer_name": "Jane Smith",
				"event_title": "Jazz Fest"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/api/v1/notifications/payment-failed", bytes.NewBufferString(tt.body))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("[%s] expected 400 Bad Request, got %d. Body: %s", tt.name, w.Code, w.Body.String())
			}

			var res dto.BaseResponse[any]
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatalf("[%s] failed to decode error response: %v", tt.name, err)
			}

			if res.Success {
				t.Errorf("[%s] expected success false, got true", tt.name)
			}
			if res.Message != "Validation failed" {
				t.Errorf("[%s] expected Message 'Validation failed', got %q", tt.name, res.Message)
			}
		})
	}
}

func TestSendCustomEmail_Success(t *testing.T) {
	router, mockSMTP, workerPool := setupTestEnvironment(t)
	defer mockSMTP.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = workerPool.Stop(ctx)
	}()

	payload := dto.CustomEmailRequest{
		To:      []string{"admin@example.com", "moderator@example.com"},
		Subject: "System Maintenance Scheduled",
		Body:    "<p>Routine database maintenance will occur at 02:00 WIB.</p>",
		IsHTML:  true,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/notifications/send-custom", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 Accepted, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res dto.BaseResponse[dto.QueuedResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	if res.Message != "Custom email notification queued for delivery" {
		t.Errorf("unexpected message: %s", res.Message)
	}
	if res.Data.JobID == "" {
		t.Errorf("expected non-empty job_id in data")
	}
	if _, err := uuid.Parse(res.Data.JobID); err != nil {
		t.Errorf("expected valid UUID for job_id, got %s: %v", res.Data.JobID, err)
	}
}

func TestSendCustomEmail_ValidationFailures(t *testing.T) {
	router, mockSMTP, workerPool := setupTestEnvironment(t)
	defer mockSMTP.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = workerPool.Stop(ctx)
	}()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed_json",
			body: `{"to": ["test@example.com"], broken`,
		},
		{
			name: "empty_recipients_list",
			body: `{
				"to": [],
				"subject": "Hello",
				"body": "Test message"
			}`,
		},
		{
			name: "invalid_recipient_email",
			body: `{
				"to": ["not-an-email"],
				"subject": "Hello",
				"body": "Test message"
			}`,
		},
		{
			name: "missing_subject",
			body: `{
				"to": ["test@example.com"],
				"body": "Test message"
			}`,
		},
		{
			name: "missing_body",
			body: `{
				"to": ["test@example.com"],
				"subject": "Hello"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/api/v1/notifications/send-custom", bytes.NewBufferString(tt.body))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("[%s] expected 400 Bad Request, got %d. Body: %s", tt.name, w.Code, w.Body.String())
			}

			var res dto.BaseResponse[any]
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatalf("[%s] failed to decode error response: %v", tt.name, err)
			}

			if res.Success {
				t.Errorf("[%s] expected success false, got true", tt.name)
			}
			if res.Message != "Validation failed" {
				t.Errorf("[%s] expected Message 'Validation failed', got %q", tt.name, res.Message)
			}
		})
	}
}
