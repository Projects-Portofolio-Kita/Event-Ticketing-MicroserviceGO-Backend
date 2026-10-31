// Package service provides business logic and integration clients for payment operations.
//
// File: booking_client.go
// Usability: Defines the Reservation model contract, BookingClient interface, and mock client for reservation verification.
package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Reservation represents the snapshot of a locked booking reservation needed for payment.
type Reservation struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TotalAmount float64
	Currency    string
	ExpiredAt   time.Time
	Status      string
}

// BookingClient defines communication with the booking service.
type BookingClient interface {
	GetReservation(reservationID uuid.UUID) (*Reservation, error)
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
		TotalAmount: 100.00,
		Currency:    "USD",
		ExpiredAt:   time.Now().Add(24 * time.Hour),
		Status:      "LOCKED",
	}, nil
}
