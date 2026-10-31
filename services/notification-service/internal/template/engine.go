package template

import (
	"bytes"
	"fmt"
	htmltmpl "html/template"
	"strings"
	texttmpl "text/template"
)

// RenderResult contains the rendered HTML and plain text alternative representations.
type RenderResult struct {
	HTML string
	Text string
}

// SeatItem represents a seat breakdown in order confirmation emails.
type SeatItem struct {
	SeatID         string
	Section        string
	Row            string
	Number         string
	PriceFormatted string
}

// OrderConfirmationData holds dynamic data for rendering the order confirmation template.
type OrderConfirmationData struct {
	BookingRef           string
	CustomerName         string
	CustomerEmail        string
	EventTitle           string
	EventDate            string
	VenueName            string
	VenueAddress         string
	TierName             string
	Seats                []SeatItem
	TotalAmountFormatted string
	Currency             string
	QRCodeBase64         string // Base64 data URL: "data:image/png;base64,..." or raw base64
	HasPDFAttachment     bool
	DownloadURL          string
	SupportEmail         string
}

// PaymentFailedData holds dynamic data for rendering the payment failed / expired reservation template.
type PaymentFailedData struct {
	BookingRef   string
	CustomerName string
	EventTitle   string
	Reason       string
	ExpiredAt    string
	RebookURL    string
	SupportEmail string
}

// EventNoticeData holds dynamic data for rendering the event cancellation / schedule change template.
type EventNoticeData struct {
	BookingRef     string
	CustomerName   string
	EventTitle     string
	NoticeType     string // e.g., "Event Cancelled", "Schedule Update", "Venue Change"
	Reason         string
	ScheduleUpdate string
	RefundInfo     string
	SupportEmail   string
}

// Engine defines the interface for rendering notification templates.
type Engine interface {
	RenderOrderConfirmation(data OrderConfirmationData) (*RenderResult, error)
	RenderPaymentFailed(data PaymentFailedData) (*RenderResult, error)
	RenderEventNotice(data EventNoticeData) (*RenderResult, error)
}

// TemplateEngine is the concrete implementation of Engine.
type TemplateEngine struct {
	orderConfirmHTML *htmltmpl.Template
	orderConfirmText *texttmpl.Template
	paymentFailHTML  *htmltmpl.Template
	paymentFailText  *texttmpl.Template
	eventNoticeHTML  *htmltmpl.Template
	eventNoticeText  *texttmpl.Template
}

// safeQR formats and marks base64 image data as safe template.URL for img src attributes.
func safeQR(s string) htmltmpl.URL {
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "data:image") {
		return htmltmpl.URL("data:image/png;base64," + s)
	}
	return htmltmpl.URL(s)
}

func safeURL(s string) htmltmpl.URL {
	return htmltmpl.URL(s)
}

// NewEngine creates and parses all HTML and plain-text templates.
func NewEngine() (*TemplateEngine, error) {
	funcMap := htmltmpl.FuncMap{
		"safeQR":  safeQR,
		"safeURL": safeURL,
	}

	// Parse Order Confirmation
	ocHTML, err := htmltmpl.New("order_confirmation_html").Funcs(funcMap).Parse(orderConfirmationHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order confirmation HTML template: %w", err)
	}
	ocText, err := texttmpl.New("order_confirmation_text").Parse(orderConfirmationTextTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order confirmation Text template: %w", err)
	}

	// Parse Payment Failed
	pfHTML, err := htmltmpl.New("payment_failed_html").Funcs(funcMap).Parse(paymentFailedHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment failed HTML template: %w", err)
	}
	pfText, err := texttmpl.New("payment_failed_text").Parse(paymentFailedTextTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment failed Text template: %w", err)
	}

	// Parse Event Notice
	enHTML, err := htmltmpl.New("event_notice_html").Funcs(funcMap).Parse(eventNoticeHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event notice HTML template: %w", err)
	}
	enText, err := texttmpl.New("event_notice_text").Parse(eventNoticeTextTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event notice Text template: %w", err)
	}

	return &TemplateEngine{
		orderConfirmHTML: ocHTML,
		orderConfirmText: ocText,
		paymentFailHTML:  pfHTML,
		paymentFailText:  pfText,
		eventNoticeHTML:  enHTML,
		eventNoticeText:  enText,
	}, nil
}

// RenderOrderConfirmation renders the order confirmation email in HTML and plain text.
func (e *TemplateEngine) RenderOrderConfirmation(data OrderConfirmationData) (*RenderResult, error) {
	if data.SupportEmail == "" {
		data.SupportEmail = "support@tickid.com"
	}
	if data.Currency == "" {
		data.Currency = "IDR"
	}

	var htmlBuf bytes.Buffer
	if err := e.orderConfirmHTML.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute order confirmation HTML template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := e.orderConfirmText.Execute(&textBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute order confirmation text template: %w", err)
	}

	return &RenderResult{
		HTML: htmlBuf.String(),
		Text: textBuf.String(),
	}, nil
}

// RenderPaymentFailed renders the payment failed / expired notification email.
func (e *TemplateEngine) RenderPaymentFailed(data PaymentFailedData) (*RenderResult, error) {
	if data.SupportEmail == "" {
		data.SupportEmail = "support@tickid.com"
	}

	var htmlBuf bytes.Buffer
	if err := e.paymentFailHTML.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute payment failed HTML template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := e.paymentFailText.Execute(&textBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute payment failed text template: %w", err)
	}

	return &RenderResult{
		HTML: htmlBuf.String(),
		Text: textBuf.String(),
	}, nil
}

// RenderEventNotice renders the event cancellation / schedule change email.
func (e *TemplateEngine) RenderEventNotice(data EventNoticeData) (*RenderResult, error) {
	if data.SupportEmail == "" {
		data.SupportEmail = "support@tickid.com"
	}
	if data.NoticeType == "" {
		data.NoticeType = "Event Update"
	}

	var htmlBuf bytes.Buffer
	if err := e.eventNoticeHTML.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute event notice HTML template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := e.eventNoticeText.Execute(&textBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute event notice text template: %w", err)
	}

	return &RenderResult{
		HTML: htmlBuf.String(),
		Text: textBuf.String(),
	}, nil
}
