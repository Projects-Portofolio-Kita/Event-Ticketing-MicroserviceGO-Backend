// Package template provides HTML and text rendering engines for email notifications.
//
// File: engine_render.go
// Usability: Implements template execution for order confirmations and payment failure notifications.
package template

import (
	"bytes"
	"fmt"
)

// RenderOrderConfirmation renders the order confirmation email in HTML and plain text.
func (e *TemplateEngine) RenderOrderConfirmation(data OrderConfirmationData) (*RenderResult, error) {
	if data.SupportEmail == "" {
		data.SupportEmail = "support@tickid.com"
	}
	if data.Currency == "" {
		data.Currency = "IDR"
	}
	var hBuf, tBuf bytes.Buffer
	if err := e.orderConfirmHTML.Execute(&hBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute order confirmation HTML template: %w", err)
	}
	if err := e.orderConfirmText.Execute(&tBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute order confirmation text template: %w", err)
	}
	return &RenderResult{HTML: hBuf.String(), Text: tBuf.String()}, nil
}

// RenderPaymentFailed renders the payment failed / expired notification email.
func (e *TemplateEngine) RenderPaymentFailed(data PaymentFailedData) (*RenderResult, error) {
	if data.SupportEmail == "" {
		data.SupportEmail = "support@tickid.com"
	}
	var hBuf, tBuf bytes.Buffer
	if err := e.paymentFailHTML.Execute(&hBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute payment failed HTML template: %w", err)
	}
	if err := e.paymentFailText.Execute(&tBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute payment failed text template: %w", err)
	}
	return &RenderResult{HTML: hBuf.String(), Text: tBuf.String()}, nil
}
