package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type SeatStatus struct {
	SeatID string `json:"seatId"`
	Status string `json:"status"`
}

type BookingClient interface {
	GetSeatStatus(eventID string) ([]SeatStatus, error)
}

type HTTPBookingClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPBookingClient() *HTTPBookingClient {
	baseURL := os.Getenv("BOOKING_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://booking-service:8083"
	}

	return &HTTPBookingClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *HTTPBookingClient) GetSeatStatus(eventID string) ([]SeatStatus, error) {
	url := fmt.Sprintf("%s/api/v1/events/%s/seat-status", c.baseURL, eventID)

	response, err := c.http.Get(url)
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
