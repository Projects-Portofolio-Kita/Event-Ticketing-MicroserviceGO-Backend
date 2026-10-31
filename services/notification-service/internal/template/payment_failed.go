// Package template provides HTML and text rendering engines for email notifications.
//
// File: payment_failed.go
// Usability: Embeds payment failure and reservation expiration alert email templates.
package template

import _ "embed"

//go:embed templates/payment_failed.html
var paymentFailedHTMLTemplate string

//go:embed templates/payment_failed.txt
var paymentFailedTextTemplate string
