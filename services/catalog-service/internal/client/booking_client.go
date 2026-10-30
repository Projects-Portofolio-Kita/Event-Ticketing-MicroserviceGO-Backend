package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
)

type SeatStatus struct {
	SeatID string `json:"seatId"`
	Status string `json:"status"`
}

// TicketStatus is how many tickets of one tier are already locked or sold.
type TicketStatus struct {
	CategoryID string `json:"categoryId"`
	Reserved   int    `json:"reserved"`
}

type BookingClient interface {
	GetSeatStatus(eventID string) ([]SeatStatus, error)
	// GetTicketStatus powers the "sisa tiket" counter of TIER events.
	GetTicketStatus(eventID string) ([]TicketStatus, error)
}

type HTTPBookingClient struct {
	baseURL      string
	serviceToken string
	http         *http.Client
}

func NewHTTPBookingClient() *HTTPBookingClient {
	baseURL := os.Getenv("BOOKING_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://booking-service:8083"
	}

	return &HTTPBookingClient{
		baseURL:      baseURL,
		serviceToken: os.Getenv("SERVICE_TOKEN"),
		http:         &http.Client{Timeout: 3 * time.Second},
	}
}

// GetSeatStatus reads booking-service's live availability feed. The feed is
// only reachable with X-Service-Token: without it every call would fail and
// seatmap would silently report every seat as available.
func (c *HTTPBookingClient) GetSeatStatus(eventID string) ([]SeatStatus, error) {
	url := fmt.Sprintf("%s/api/v1/events/%s/seat-status", c.baseURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.serviceToken != "" {
		req.Header.Set(authz.ServiceHeader, c.serviceToken)
	}

	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("booking-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("booking-service returned status %d", response.StatusCode)
	}

	var body struct {
		Seats []SeatStatus `json:"seats"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode seat status: %w", err)
	}

	return body.Seats, nil
}

// GetTicketStatus reads booking-service's tier availability feed (quantity
// reserved per category). Like the seat feed it requires X-Service-Token.
func (c *HTTPBookingClient) GetTicketStatus(eventID string) ([]TicketStatus, error) {
	url := fmt.Sprintf("%s/api/v1/events/%s/ticket-status", c.baseURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.serviceToken != "" {
		req.Header.Set(authz.ServiceHeader, c.serviceToken)
	}

	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("booking-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("booking-service returned status %d", response.StatusCode)
	}

	var body struct {
		Tickets []TicketStatus `json:"tickets"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode ticket status: %w", err)
	}

	return body.Tickets, nil
}
