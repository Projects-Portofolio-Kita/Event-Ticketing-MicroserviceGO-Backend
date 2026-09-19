package dto

import (
	"time"

	"github.com/google/uuid"
)

type SeatItemDTO struct {
	SeatID  uuid.UUID `json:"seatId" binding:"required"`
	Section string    `json:"section" binding:"required"`
	Row     string    `json:"row" binding:"required"`
	Number  string    `json:"number" binding:"required"`
	Price   float64   `json:"price" binding:"required,gt=0"`
}

type CreateReservationRequest struct {
	UserID   uuid.UUID     `json:"userId" binding:"required"`
	EventID  uuid.UUID     `json:"eventId" binding:"required"`
	Seats    []SeatItemDTO `json:"seats" binding:"required,min=1,dive"`
	Currency string        `json:"currency"`
}

type SeatResponseDTO struct {
	SeatID  uuid.UUID `json:"seatId"`
	Section string    `json:"section"`
	Row     string    `json:"row"`
	Number  string    `json:"number"`
	Price   float64   `json:"price"`
}

type ReservationResponse struct {
	ReservationID uuid.UUID         `json:"reservationId"`
	UserID        uuid.UUID         `json:"userId"`
	EventID       uuid.UUID         `json:"eventId"`
	BookingRef    string            `json:"bookingRef"`
	Status        string            `json:"status"`
	ExpiresAt     time.Time         `json:"expiresAt"`
	TotalAmount   float64           `json:"totalAmount"`
	Currency      string            `json:"currency"`
	Seats         []SeatResponseDTO `json:"seats"`
	CreatedAt     time.Time         `json:"createdAt"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
