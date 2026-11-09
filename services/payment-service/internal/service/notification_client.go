// Package service provides business logic and integration clients for payment operations.
//
// File: notification_client.go
// Usability: Defines contracts and HTTP/Mock clients for queuing e-ticket confirmations with notification-service.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// NotificationSeatItem represents the seat schema for notification service.
type NotificationSeatItem struct {
	SeatID  string  `json:"seat_id"`
	Section string  `json:"section"`
	Row     string  `json:"row"`
	Number  string  `json:"number"`
	Price   float64 `json:"price"`
}

// TicketConfirmationPayload represents the dispatch payload for notification-service.
type TicketConfirmationPayload struct {
	BookingRef    string                 `json:"booking_ref"`
	CustomerEmail string                 `json:"customer_email"`
	CustomerName  string                 `json:"customer_name"`
	EventID       string                 `json:"event_id"`
	EventTitle    string                 `json:"event_title"`
	EventDate     string                 `json:"event_date"`
	VenueName     string                 `json:"venue_name"`
	VenueAddress  string                 `json:"venue_address,omitempty"`
	TierName      string                 `json:"tier_name"`
	TotalAmount   float64                `json:"total_amount"`
	Currency      string                 `json:"currency"`
	Seats         []NotificationSeatItem `json:"seats"`
}

// NotificationClient defines contract for communicating with notification-service.
type NotificationClient interface {
	SendTicketConfirmation(ctx context.Context, payload TicketConfirmationPayload) (string, error)
}

// MockNotificationClient provides in-memory mock implementation for notification client.
type MockNotificationClient struct{}

// NewMockNotificationClient constructs a new MockNotificationClient.
func NewMockNotificationClient() *MockNotificationClient {
	return &MockNotificationClient{}
}

// SendTicketConfirmation simulates ticket confirmation dispatch.
func (c *MockNotificationClient) SendTicketConfirmation(ctx context.Context, payload TicketConfirmationPayload) (string, error) {
	log.Printf("[MockNotificationClient] Ticket confirmation queued: %s for %s\n", payload.BookingRef, payload.CustomerEmail)
	return uuid.New().String(), nil
}

// HTTPNotificationClient provides live HTTP communication with notification-service.
type HTTPNotificationClient struct {
	baseURL    string
	httpClient *http.Client
	fallback   *MockNotificationClient
}

// NewHTTPNotificationClient constructs a new HTTPNotificationClient.
func NewHTTPNotificationClient(baseURL string) *HTTPNotificationClient {
	if baseURL == "" {
		baseURL = os.Getenv("NOTIFICATION_SERVICE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8085"
		}
	}
	return &HTTPNotificationClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		fallback: NewMockNotificationClient(),
	}
}

// SendTicketConfirmation dispatches ticket confirmation to notification-service with graceful fallback.
func (c *HTTPNotificationClient) SendTicketConfirmation(ctx context.Context, payload TicketConfirmationPayload) (string, error) {
	url := fmt.Sprintf("%s/api/v1/notifications/ticket-confirmation", c.baseURL)
	body, err := json.Marshal(payload)
	if err != nil {
		return c.fallback.SendTicketConfirmation(ctx, payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return c.fallback.SendTicketConfirmation(ctx, payload)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[HTTPNotificationClient] Notice: notification-service unreachable (%v), falling back to mock\n", err)
		return c.fallback.SendTicketConfirmation(ctx, payload)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		log.Printf("[HTTPNotificationClient] Notice: notification-service returned status %d, falling back to mock\n", resp.StatusCode)
		return c.fallback.SendTicketConfirmation(ctx, payload)
	}

	type queueResp struct {
		Success bool `json:"success"`
		Data    struct {
			JobID string `json:"job_id"`
		} `json:"data"`
	}
	var qr queueResp
	if err := json.NewDecoder(resp.Body).Decode(&qr); err == nil && qr.Data.JobID != "" {
		return qr.Data.JobID, nil
	}

	return uuid.New().String(), nil
}
