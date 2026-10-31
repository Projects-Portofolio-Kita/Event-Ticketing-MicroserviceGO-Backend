// Package template provides HTML and text rendering engines for email notifications.
//
// File: event_notice.go
// Usability: Embeds event update, reschedule, and cancellation announcement email templates.
package template

import _ "embed"

//go:embed templates/event_notice.html
var eventNoticeHTMLTemplate string

//go:embed templates/event_notice.txt
var eventNoticeTextTemplate string
