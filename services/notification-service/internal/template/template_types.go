// Package template provides HTML and text rendering engines for email notifications.
//
// File: template_types.go
// Usability: Declares RenderResult, SeatItem, data models (OrderConfirmationData, PaymentFailedData, EventNoticeData), and Engine interface.
package template

// RenderResult contains the rendered HTML and plain text alternative representations.
type RenderResult struct {
	HTML string
	Text string
}

// SeatItem represents a seat breakdown in order confirmation emails.
type SeatItem struct {
	SeatID, Section, Row, Number, PriceFormatted string
}

// OrderConfirmationData holds dynamic data for rendering the order confirmation template.
type OrderConfirmationData struct {
	BookingRef, CustomerName, CustomerEmail, EventTitle, EventDate, VenueName, VenueAddress string
	TierName, TotalAmountFormatted, Currency, QRCodeBase64, DownloadURL, SupportEmail       string
	Seats                                                                                   []SeatItem
	HasPDFAttachment                                                                        bool
}

// PaymentFailedData holds dynamic data for rendering the payment failed / expired reservation template.
type PaymentFailedData struct {
	BookingRef, CustomerName, EventTitle, Reason, ExpiredAt, RebookURL, SupportEmail string
}

// EventNoticeData holds dynamic data for rendering the event cancellation / schedule change template.
type EventNoticeData struct {
	BookingRef, CustomerName, EventTitle, NoticeType, Reason, ScheduleUpdate, RefundInfo, SupportEmail string
}

// Engine defines the interface for rendering notification templates.
type Engine interface {
	RenderOrderConfirmation(data OrderConfirmationData) (*RenderResult, error)
	RenderPaymentFailed(data PaymentFailedData) (*RenderResult, error)
	RenderEventNotice(data EventNoticeData) (*RenderResult, error)
}
