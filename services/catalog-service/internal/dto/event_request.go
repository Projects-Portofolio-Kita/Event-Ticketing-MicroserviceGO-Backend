package dto

// CreateEventCategory is one priced category (seat section, or a quota-based
// ticket tier such as VIP/VVIP/Regular) of a new event. For TIER events the
// service generates no seats: Quota is how many tickets may be sold.
type CreateEventCategory struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Price       float64 `json:"price" binding:"required,min=0"`
	Quota       int     `json:"quota" binding:"omitempty,min=1,max=1000000"`
	Description string  `json:"description" binding:"omitempty,max=500"`
}

// CreateEventRequest posts a new event on behalf of the signed-in user.
// Code is optional — it is derived from the name when empty. EventDate accepts
// RFC3339 ("2026-06-15T19:30:00+07:00") or a plain date ("2026-06-15").
//
// SaleMode: "SEATED" (default) generates a numbered seat map, "TIER" sells
// quota-based tickets only (every category must carry a Quota).
type CreateEventRequest struct {
	Code                string                `json:"code" binding:"omitempty,max=50"`
	Name                string                `json:"name" binding:"required,max=255"`
	Venue               string                `json:"venue" binding:"required,max=255"`
	EventDate           string                `json:"eventDate" binding:"required"`
	City                string                `json:"city" binding:"omitempty,max=100"`
	Category            string                `json:"category" binding:"omitempty,max=50"`
	Description         string                `json:"description" binding:"omitempty,max=2000"`
	PosterURL           string                `json:"posterUrl" binding:"omitempty,max=512"`
	SaleMode            string                `json:"saleMode" binding:"omitempty,max=10"`
	SeatRows            int                   `json:"seatRows" binding:"omitempty,min=1,max=26"`
	SeatsPerRow         int                   `json:"seatsPerRow" binding:"omitempty,min=2,max=60"`
	LeftBlockSize       int                   `json:"leftBlockSize" binding:"omitempty,min=1,max=30"`
	PayoutBankName      string                `json:"payoutBankName" binding:"omitempty,max=100"`
	PayoutAccountNumber string                `json:"payoutAccountNumber" binding:"omitempty,max=50"`
	PayoutAccountName   string                `json:"payoutAccountName" binding:"omitempty,max=100"`
	Categories          []CreateEventCategory `json:"categories" binding:"required,min=1,max=10,dive"`
}
