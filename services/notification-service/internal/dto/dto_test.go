package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
)

func TestBaseResponse_Success(t *testing.T) {
	data := dto.QueuedResponse{
		JobID:  "test-job-123",
		Status: "queued",
	}

	resp := dto.SuccessResponse(data, "Notification queued for delivery")
	if !resp.Success {
		t.Errorf("expected Success to be true")
	}
	if resp.Message != "Notification queued for delivery" {
		t.Errorf("expected message 'Notification queued for delivery', got '%s'", resp.Message)
	}
	if resp.Data.JobID != "test-job-123" {
		t.Errorf("expected JobID 'test-job-123', got '%s'", resp.Data.JobID)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal BaseResponse: %v", err)
	}
	if len(bytes) == 0 {
		t.Fatal("empty json output")
	}
}

func TestBaseResponse_Error(t *testing.T) {
	resp := dto.ErrorResponse[any]("Validation failed", "Invalid payload")
	if resp.Success {
		t.Errorf("expected Success to be false")
	}
	if resp.Error != "Validation failed" {
		t.Errorf("expected error 'Validation failed', got '%s'", resp.Error)
	}
	if resp.Message != "Invalid payload" {
		t.Errorf("expected message 'Invalid payload', got '%s'", resp.Message)
	}
}

func TestTicketConfirmationRequest_JSON(t *testing.T) {
	reqJSON := `{
		"booking_ref": "BK-20261018-001",
		"customer_email": "john.doe@example.com",
		"customer_name": "John Doe",
		"event_id": "evt-123",
		"event_title": "Coldplay Music of the Spheres",
		"event_date": "2026-11-15 19:30",
		"venue_name": "Gelora Bung Karno",
		"venue_address": "Jakarta, Indonesia",
		"tier_name": "VIP Cat 1",
		"total_amount": 2500000.0,
		"currency": "IDR",
		"seats": [
			{
				"seat_id": "seat-A-1",
				"section": "VIP",
				"row": "A",
				"number": "1",
				"price": 2500000.0
			}
		]
	}`

	var req dto.TicketConfirmationRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		t.Fatalf("failed to unmarshal TicketConfirmationRequest: %v", err)
	}

	if req.BookingRef != "BK-20261018-001" {
		t.Errorf("expected BookingRef 'BK-20261018-001', got '%s'", req.BookingRef)
	}
	if len(req.Seats) != 1 {
		t.Fatalf("expected 1 seat, got %d", len(req.Seats))
	}
	if req.Seats[0].SeatID != "seat-A-1" {
		t.Errorf("expected SeatID 'seat-A-1', got '%s'", req.Seats[0].SeatID)
	}
}

func TestPaymentFailedRequest_JSON(t *testing.T) {
	reqJSON := `{
		"booking_ref": "BK-20261018-002",
		"customer_email": "jane@example.com",
		"customer_name": "Jane Doe",
		"event_title": "Taylor Swift Eras Tour",
		"reason": "Payment expired after 15 minutes",
		"expired_at": "2026-10-18T15:00:00Z"
	}`

	var req dto.PaymentFailedRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		t.Fatalf("failed to unmarshal PaymentFailedRequest: %v", err)
	}

	if req.BookingRef != "BK-20261018-002" {
		t.Errorf("expected BookingRef 'BK-20261018-002', got '%s'", req.BookingRef)
	}
	if req.Reason != "Payment expired after 15 minutes" {
		t.Errorf("unexpected reason: %s", req.Reason)
	}
}

func TestCustomEmailRequest_JSON(t *testing.T) {
	reqJSON := `{
		"to": ["user1@example.com", "user2@example.com"],
		"subject": "Important Event Update",
		"body": "<p>Event gate opens at 17:00</p>",
		"is_html": true
	}`

	var req dto.CustomEmailRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		t.Fatalf("failed to unmarshal CustomEmailRequest: %v", err)
	}

	if len(req.To) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(req.To))
	}
	if !req.IsHTML {
		t.Errorf("expected IsHTML true")
	}
}
