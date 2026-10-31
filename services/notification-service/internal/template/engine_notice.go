// Package template provides HTML and text rendering engines for email notifications.
//
// File: engine_notice.go
// Usability: Implements template execution for event cancellation, update, and schedule change emails.
package template

import (
	"bytes"
	"fmt"
)

// RenderEventNotice renders the event cancellation / schedule change email.
func (e *TemplateEngine) RenderEventNotice(data EventNoticeData) (*RenderResult, error) {
	if data.SupportEmail == "" {
		data.SupportEmail = "support@tickid.com"
	}
	if data.NoticeType == "" {
		data.NoticeType = "Event Update"
	}
	var hBuf, tBuf bytes.Buffer
	if err := e.eventNoticeHTML.Execute(&hBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute event notice HTML template: %w", err)
	}
	if err := e.eventNoticeText.Execute(&tBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute event notice text template: %w", err)
	}
	return &RenderResult{HTML: hBuf.String(), Text: tBuf.String()}, nil
}
