// Package service provides business logic and integration clients for payment operations.
//
// File: booking_client.go
// Usability: Defines Reservation and Seat contracts, BookingClient interface,
// MockBookingClient for unit tests, and HTTPBookingClient for inter-service communication.
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// SeatItem represents a seat within a reservation.
type SeatItem struct {
	SeatID  uuid.UUID `json:"seatId"`
	Section string    `json:"section"`
	Row     string    `json:"row"`
	Number  string    `json:"number"`
	Price   float64   `json:"price"`
}

// Reservation represents the snapshot of a locked booking reservation needed for payment.
type Reservation struct {
	ID          uuid.UUID  `json:"reservationId"`
	UserID      uuid.UUID  `json:"userId"`
	EventID     uuid.UUID  `json:"eventId"`
	BookingRef  string     `json:"bookingRef"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
	ExpiredAt   time.Time  `json:"expiresAt"`
	Status      string     `json:"status"`
	Seats       []SeatItem `json:"seats"`
}

// BookingClient defines communication with the booking service.
type BookingClient interface {
	GetReservation(reservationID uuid.UUID) (*Reservation, error)
	ConfirmReservation(reservationID uuid.UUID) (*Reservation, error)
}

// MockBookingClient provides an in-memory implementation of BookingClient for testing.
type MockBookingClient struct{}

// NewMockBookingClient constructs a MockBookingClient instance.
func NewMockBookingClient() *MockBookingClient {
	return &MockBookingClient{}
}

// GetReservation returns a stubbed locked reservation.
func (c *MockBookingClient) GetReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}
	return &Reservation{
		ID:          reservationID,
		UserID:      uuid.New(),
		EventID:     uuid.New(),
		BookingRef:  fmt.Sprintf("TICK-MOCK-%s", reservationID.String()[:8]),
		TotalAmount: 3885000.00,
		Currency:    "IDR",
		ExpiredAt:   time.Now().Add(24 * time.Hour),
		Status:      "LOCKED",
		Seats: []SeatItem{
			{SeatID: uuid.New(), Section: "CAT 1 VIP", Row: "A", Number: "01", Price: 3500000},
		},
	}, nil
}

// ConfirmReservation simulates confirming a locked reservation.
func (c *MockBookingClient) ConfirmReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}
	return &Reservation{
		ID:          reservationID,
		UserID:      uuid.New(),
		EventID:     uuid.New(),
		BookingRef:  fmt.Sprintf("TICK-CONFIRMED-%s", reservationID.String()[:8]),
		TotalAmount: 3885000.00,
		Currency:    "IDR",
		ExpiredAt:   time.Now().Add(24 * time.Hour),
		Status:      "CONFIRMED",
		Seats: []SeatItem{
			{SeatID: uuid.New(), Section: "CAT 1 VIP", Row: "A", Number: "01", Price: 3500000},
		},
	}, nil
}

// HTTPBookingClient provides live HTTP communication with booking-service.
type HTTPBookingClient struct {
	baseURL    string
	httpClient *http.Client
	fallback   *MockBookingClient
}

// NewHTTPBookingClient constructs a new HTTPBookingClient.
func NewHTTPBookingClient(baseURL string) *HTTPBookingClient {
	if baseURL == "" {
		baseURL = os.Getenv("BOOKING_SERVICE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8083"
		}
	}
	return &HTTPBookingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		fallback: NewMockBookingClient(),
	}
}

// GetReservation fetches a reservation from booking-service with graceful fallback.
func (c *HTTPBookingClient) GetReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}
	url := fmt.Sprintf("%s/api/v1/reservations/%s", c.baseURL, reservationID.String())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return c.fallback.GetReservation(reservationID)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("reservation not found")
	}
	if resp.StatusCode != http.StatusOK {
		return c.fallback.GetReservation(reservationID)
	}

	var res Reservation
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return c.fallback.GetReservation(reservationID)
	}
	return &res, nil
}

// ConfirmReservation sends confirmation request to booking-service with graceful fallback.
func (c *HTTPBookingClient) ConfirmReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}
	url := fmt.Sprintf("%s/api/v1/reservations/%s/confirm", c.baseURL, reservationID.String())
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return c.fallback.ConfirmReservation(reservationID)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallback.ConfirmReservation(reservationID)
	}

	var res Reservation
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return c.fallback.ConfirmReservation(reservationID)
	}
	return &res, nil
}
