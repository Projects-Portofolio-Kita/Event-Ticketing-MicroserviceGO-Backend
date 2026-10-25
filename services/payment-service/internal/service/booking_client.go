package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
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
		http:         &http.Client{Timeout: 5 * time.Second},
	}
}

// request builds an outgoing call to booking-service. Those endpoints accept
// either a user token or X-Service-Token; this is a service-to-service call,
// so it identifies the *service*, never a user — ownership of the reservation
// is verified against the caller's own identity before payment proceeds.
func (c *HTTPBookingClient) request(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	if c.serviceToken != "" {
		req.Header.Set(authz.ServiceHeader, c.serviceToken)
	}
	return req, nil
}

func (c *HTTPBookingClient) GetReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, fmt.Errorf("%w: reservation id", ErrInvalidInput)
	}

	url := fmt.Sprintf("%s/api/v1/reservations/%s", c.baseURL, reservationID.String())

	req, err := c.request(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("booking-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, ErrReservationNotFound
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

	req, err := c.request(http.MethodPost, url)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("booking-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to confirm reservation (status %d)", response.StatusCode)
	}

	return nil
}
