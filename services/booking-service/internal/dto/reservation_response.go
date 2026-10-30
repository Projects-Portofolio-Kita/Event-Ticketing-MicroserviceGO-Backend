package dto

import "time"

type SeatItemResponse struct {
	SeatID  string  `json:"seatId"`
	Section string  `json:"section"`
	Row     string  `json:"row"`
	Number  string  `json:"number"`
	Price   float64 `json:"price"`
}

// TicketItemResponse is one tier line of a TIER reservation.
type TicketItemResponse struct {
	CategoryID   string  `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unitPrice"`
	Subtotal     float64 `json:"subtotal"`
}

type ReservationResponse struct {
	ReservationID string               `json:"reservationId"`
	UserID        string               `json:"userId"`
	EventID       string               `json:"eventId"`
	BookingRef    string               `json:"bookingRef"`
	Status        string               `json:"status"`
	Mode          string               `json:"mode"`
	ExpiresAt     time.Time            `json:"expiresAt"`
	TotalAmount   float64              `json:"totalAmount"`
	Currency      string               `json:"currency"`
	Seats         []SeatItemResponse   `json:"seats"`
	Items         []TicketItemResponse `json:"items"`
	CreatedAt     time.Time            `json:"createdAt"`
}
