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

type bookingReservationDTO struct {
	ReservationID string    `json:"reservationId"`
	UserID        string    `json:"userId"`
	Status        string    `json:"status"`
	ExpiresAt     time.Time `json:"expiresAt"`
	TotalAmount   float64   `json:"totalAmount"`
	Currency      string    `json:"currency"`
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
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPBookingClient) GetReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}

	url := fmt.Sprintf("%s/api/v1/reservations/%s", c.baseURL, reservationID.String())

	response, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("booking-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, errors.New("reservation not found in booking-service")
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("booking-service returned status %d", response.StatusCode)
	}

	var dto bookingReservationDTO
	if err := json.NewDecoder(response.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf("failed to decode booking response: %w", err)
	}

	userID, err := uuid.Parse(dto.UserID)
	if err != nil {
		return nil, errors.New("invalid user id from booking-service")
	}

	return &Reservation{
		ID:          reservationID,
		UserID:      userID,
		TotalAmount: dto.TotalAmount,
		Currency:    dto.Currency,
		ExpiredAt:   dto.ExpiresAt,
		Status:      dto.Status,
	}, nil
}

func (c *HTTPBookingClient) ConfirmReservation(reservationID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/reservations/%s/confirm", c.baseURL, reservationID.String())

	response, err := c.http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("booking-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to confirm reservation (status %d)", response.StatusCode)
	}

	return nil
}
