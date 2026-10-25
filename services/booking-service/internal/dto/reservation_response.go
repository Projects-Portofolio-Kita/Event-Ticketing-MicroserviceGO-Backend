package dto

import "time"

type SeatItemResponse struct {
	SeatID  string  `json:"seatId"`
	Section string  `json:"section"`
	Row     string  `json:"row"`
	Number  string  `json:"number"`
	Price   float64 `json:"price"`
}

type ReservationResponse struct {
	ReservationID string             `json:"reservationId"`
	UserID        string             `json:"userId"`
	EventID       string             `json:"eventId"`
	BookingRef    string             `json:"bookingRef"`
	Status        string             `json:"status"`
	ExpiresAt     time.Time          `json:"expiresAt"`
	TotalAmount   float64            `json:"totalAmount"`
	Currency      string             `json:"currency"`
	Seats         []SeatItemResponse `json:"seats"`
	CreatedAt     time.Time          `json:"createdAt"`
}
