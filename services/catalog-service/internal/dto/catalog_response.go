package dto

type EventResponse struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Venue string `json:"venue"`
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
