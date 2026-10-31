package dto

// SeatInfoDTO represents individual seat information within a booking
type SeatInfoDTO struct {
	SeatID  string  `json:"seat_id" binding:"required"`
	Section string  `json:"section,omitempty"`
	Row     string  `json:"row,omitempty"`
	Number  string  `json:"number" binding:"required"`
	Price   float64 `json:"price"`
}

// TicketConfirmationRequest represents the payload for order confirmation & E-ticket delivery
type TicketConfirmationRequest struct {
	BookingRef    string        `json:"booking_ref" binding:"required"`
	CustomerEmail string        `json:"customer_email" binding:"required,email"`
	CustomerName  string        `json:"customer_name" binding:"required"`
	EventID       string        `json:"event_id" binding:"required"`
	EventTitle    string        `json:"event_title" binding:"required"`
	EventDate     string        `json:"event_date" binding:"required"`
	VenueName     string        `json:"venue_name" binding:"required"`
	VenueAddress  string        `json:"venue_address,omitempty"`
	TierName      string        `json:"tier_name" binding:"required"`
	TotalAmount   float64       `json:"total_amount" binding:"required"`
	Currency      string        `json:"currency,omitempty"`
	Seats         []SeatInfoDTO `json:"seats" binding:"required,dive"`
}

// PaymentFailedRequest represents the payload when a payment fails or reservation expires
type PaymentFailedRequest struct {
	BookingRef    string `json:"booking_ref" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	CustomerName  string `json:"customer_name" binding:"required"`
	EventTitle    string `json:"event_title" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	ExpiredAt     string `json:"expired_at,omitempty"`
}

// CustomEmailRequest represents the payload for ad-hoc or broadcast notifications
type CustomEmailRequest struct {
	To      []string `json:"to" binding:"required,min=1,dive,email"`
	Subject string   `json:"subject" binding:"required"`
	Body    string   `json:"body" binding:"required"`
	IsHTML  bool     `json:"is_html"`
}
