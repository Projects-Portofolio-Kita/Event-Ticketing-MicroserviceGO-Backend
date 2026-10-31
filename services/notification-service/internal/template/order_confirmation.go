// Package template provides HTML and text rendering engines for email notifications.
//
// File: order_confirmation.go
// Usability: Embeds order confirmation HTML and plaintext email templates for successful bookings.
package template

import _ "embed"

//go:embed templates/order_confirmation.html
var orderConfirmationHTMLTemplate string

//go:embed templates/order_confirmation.txt
var orderConfirmationTextTemplate string
