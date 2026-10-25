package dto

type SeatItemRequest struct {
	SeatID  string  `json:"seatId" binding:"required"`
	Section string  `json:"section" binding:"required"`
	Row     string  `json:"row" binding:"required"`
	Number  string  `json:"number" binding:"required"`
	Price   float64 `json:"price" binding:"required"`
}

type CreateReservationRequest struct {
	UserID   string            `json:"userId" binding:"required"`
	EventID  string            `json:"eventId" binding:"required"`
	Seats    []SeatItemRequest `json:"seats" binding:"required,dive"`
	Currency string            `json:"currency"`
}
