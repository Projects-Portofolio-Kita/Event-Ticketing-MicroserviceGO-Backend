package dto

// EventResponse is the discovery/detail shape returned to clients.
type EventResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Venue       string `json:"venue"`
	EventDate   string `json:"eventDate"`
	City        string `json:"city"`
	Category    string `json:"category"`
	SaleMode    string `json:"saleMode"`
	Description string `json:"description,omitempty"`
	PosterURL   string `json:"posterUrl,omitempty"`
	// MinPrice is the cheapest category price; 0 when no categories exist.
	MinPrice float64 `json:"minPrice"`
	// TicketTypeCount is how many categories/tiers the event sells.
	TicketTypeCount int    `json:"ticketTypeCount"`
	CreatedBy       string `json:"createdBy,omitempty"`
}

type SeatResponse struct {
	ID       string  `json:"id"`
	SeatID   string  `json:"seatId"`
	Row      string  `json:"row"`
	Number   int     `json:"number"`
	Status   string  `json:"status"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}

type RowResponse struct {
	RowName    string         `json:"rowName"`
	LeftSeats  []SeatResponse `json:"leftSeats"`
	RightSeats []SeatResponse `json:"rightSeats"`
}

type CategoryResponse struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Subtitle string        `json:"subtitle"`
	DotColor string        `json:"dotColor"`
	Price    float64       `json:"price"`
	Rows     []RowResponse `json:"rows"`
}

// TicketTypeResponse is one quota-based ticket tier with live availability.
type TicketTypeResponse struct {
	CategoryID  string  `json:"categoryId"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price"`
	// Quota is the total stock configured by the poster.
	Quota int `json:"quota"`
	// Remaining = Quota minus locked/confirmed tickets (0 when booking is down).
	Remaining int `json:"remaining"`
}

// TicketTypeListResponse is GET /events/:id/ticket-types.
type TicketTypeListResponse struct {
	SaleMode    string               `json:"saleMode"`
	TicketTypes []TicketTypeResponse `json:"ticketTypes"`
}

// TierQuota is the internal (service-to-service) stock feed booking-service
// reads to enforce quotas server-side. Prices come from here too, so a client
// can never dictate what a ticket costs.
type TierQuota struct {
	CategoryID string  `json:"categoryId"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Quota      int     `json:"quota"`
}

// TierQuotaListResponse is GET /events/:id/tier-quotas (X-Service-Token).
type TierQuotaListResponse struct {
	Tiers []TierQuota `json:"tiers"`
}

// CityEntry is one selectable destination in the location picker.
type CityEntry struct {
	City  string `json:"city"`
	Count int    `json:"count"`
}

// CityListResponse is GET /events/cities.
type CityListResponse struct {
	Cities []CityEntry `json:"cities"`
}
