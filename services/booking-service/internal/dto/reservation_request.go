package dto

type SeatItemRequest struct {
	SeatID  string  `json:"seatId" binding:"required"`
	Section string  `json:"section" binding:"required"`
	Row     string  `json:"row" binding:"required"`
	Number  string  `json:"number" binding:"required"`
	Price   float64 `json:"price" binding:"required"`
}

// TicketItemRequest is one line of a TIER reservation. Only the tier and the
// wanted quantity come from the client: price and stock live server-side.
type TicketItemRequest struct {
	CategoryID string `json:"categoryId" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1,max=10"`
}

// CreateReservationRequest books seats (SEATED) or ticket tiers (TIER).
//
// UserID is accepted for backwards compatibility but always ignored: identity
// comes from the verified access token. Mode defaults from the payload (items
// present → TIER, otherwise SEATED) and is validated against its content, so a
// client can never post tier items on a seated event by accident.
//
// Prices in Seats are informational only for SEATED flows (kept as the client
// rendered them, same as before); TIER prices always come from catalog-service.
type CreateReservationRequest struct {
	UserID   string              `json:"userId"`
	EventID  string              `json:"eventId" binding:"required"`
	Mode     string              `json:"mode" binding:"omitempty,max=10"`
	Seats    []SeatItemRequest   `json:"seats"`
	Items    []TicketItemRequest `json:"items"`
	Currency string              `json:"currency"`
}
