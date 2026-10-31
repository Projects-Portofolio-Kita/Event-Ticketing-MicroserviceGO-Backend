// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_types.go
// Usability: Declares the TicketPDFData input model and PDFGenerator interface contracts.
package pdf

// TicketPDFData holds the comprehensive event, attendee, and seating information
// required to render a branded TickId e-ticket.
type TicketPDFData struct {
	BookingRef   string `json:"booking_ref"`
	EventTitle   string `json:"event_title"`
	EventDate    string `json:"event_date"`
	VenueName    string `json:"venue_name"`
	VenueAddress string `json:"venue_address"`
	AttendeeName string `json:"attendee_name"`
	SeatDetails  string `json:"seat_details"`
	TierName     string `json:"tier_name"`
	TotalAmount  string `json:"total_amount"`
	Currency     string `json:"currency,omitempty"`
	QRData       string `json:"qr_data,omitempty"`
	QRBytes      []byte `json:"-"`
}

// PDFGenerator defines the interface for generating e-ticket PDF documents.
type PDFGenerator interface {
	GenerateTicket(data TicketPDFData) ([]byte, error)
	GenerateMultiTicket(data []TicketPDFData) ([]byte, error)
}