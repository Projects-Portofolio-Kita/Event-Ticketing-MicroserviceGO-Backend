package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/google/uuid"
)

func TestMockBookingClient_GetReservation(t *testing.T) {
	client := service.NewMockBookingClient()
	resID := uuid.New()

	res, err := client.GetReservation(resID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.ID != resID {
		t.Errorf("expected res ID %s, got %s", resID, res.ID)
	}
	if res.Status != "LOCKED" {
		t.Errorf("expected status LOCKED, got %s", res.Status)
	}
	if res.TotalAmount <= 0 {
		t.Errorf("expected positive total amount, got %f", res.TotalAmount)
	}
}

func TestMockBookingClient_ConfirmReservation(t *testing.T) {
	client := service.NewMockBookingClient()
	resID := uuid.New()

	res, err := client.ConfirmReservation(resID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", res.Status)
	}
}

func TestHTTPBookingClient_LiveAndFallback(t *testing.T) {
	resID := uuid.New()
	mockRes := service.Reservation{
		ID:          resID,
		UserID:      uuid.New(),
		EventID:     uuid.New(),
		BookingRef:  "TICK-TEST-1234",
		TotalAmount: 2500000,
		Currency:    "IDR",
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Status:      "LOCKED",
		Seats: []service.SeatItem{
			{SeatID: uuid.New(), Section: "CAT 1 VIP", Row: "A", Number: "01", Price: 2500000},
		},
	}

	// Spin up simulated booking-service server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/reservations/"+resID.String() {
			json.NewEncoder(w).Encode(mockRes)
			return
		}
		if r.URL.Path == "/api/v1/reservations/"+resID.String()+"/confirm" {
			confirmed := mockRes
			confirmed.Status = "CONFIRMED"
			json.NewEncoder(w).Encode(confirmed)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := service.NewHTTPBookingClient(server.URL)

	// Test GET reservation
	res, err := client.GetReservation(resID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}
	if res.BookingRef != "TICK-TEST-1234" {
		t.Errorf("expected booking ref TICK-TEST-1234, got %s", res.BookingRef)
	}

	// Test CONFIRM reservation
	confRes, err := client.ConfirmReservation(resID)
	if err != nil {
		t.Fatalf("ConfirmReservation failed: %v", err)
	}
	if confRes.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", confRes.Status)
	}

	// Test fallback when server unreachable
	deadClient := service.NewHTTPBookingClient("http://127.0.0.1:59999")
	fallbackRes, err := deadClient.GetReservation(resID)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if fallbackRes == nil || fallbackRes.Status != "LOCKED" {
		t.Errorf("expected fallback reservation in LOCKED state")
	}
}

func TestHTTPNotificationClient_LiveAndFallback(t *testing.T) {
	receivedPayload := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/notifications/ticket-confirmation" {
			receivedPayload = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{"success":true,"message":"queued","data":{"job_id":"job-test-uuid"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := service.NewHTTPNotificationClient(server.URL)
	payload := service.TicketConfirmationPayload{
		BookingRef:    "TICK-TEST-9999",
		CustomerEmail: "tester@tickid.com",
		CustomerName:  "Test User",
		EventID:       "evt-123",
		EventTitle:    "Rock Festival 2026",
		EventDate:     "2026-11-20 20:00 WIB",
		VenueName:     "GBK Jakarta",
		TierName:      "CAT 1",
		TotalAmount:   1500000,
		Currency:      "IDR",
	}

	jobID, err := client.SendTicketConfirmation(context.Background(), payload)
	if err != nil {
		t.Fatalf("SendTicketConfirmation failed: %v", err)
	}
	if !receivedPayload {
		t.Errorf("server never received ticket-confirmation POST")
	}
	if jobID != "job-test-uuid" {
		t.Errorf("expected job ID 'job-test-uuid', got %s", jobID)
	}

	// Test fallback when notification server unreachable
	deadClient := service.NewHTTPNotificationClient("http://127.0.0.1:59999")
	fallbackJobID, err := deadClient.SendTicketConfirmation(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if fallbackJobID == "" {
		t.Errorf("expected non-empty fallback job ID")
	}
}
