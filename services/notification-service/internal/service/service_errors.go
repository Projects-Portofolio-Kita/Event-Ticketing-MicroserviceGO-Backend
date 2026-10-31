// Package service implements notification business logic and background queue dispatching.
//
// File: service_errors.go
// Usability: Declares domain validation and initialization errors for notification workflows.
package service

import "errors"

var (
	// ErrInvalidEmail indicates that an email address is empty or malformed
	ErrInvalidEmail = errors.New("invalid or empty email address")

	// ErrEmptyBookingRef indicates that booking_ref is missing
	ErrEmptyBookingRef = errors.New("booking reference cannot be empty")

	// ErrEmptyCustomerName indicates that customer_name is missing
	ErrEmptyCustomerName = errors.New("customer name cannot be empty")

	// ErrEmptyEventTitle indicates that event_title is missing
	ErrEmptyEventTitle = errors.New("event title cannot be empty")

	// ErrEmptySeats indicates that seats list is empty
	ErrEmptySeats = errors.New("at least one seat is required for ticket confirmation")

	// ErrEmptyRecipients indicates that recipient list is empty
	ErrEmptyRecipients = errors.New("recipients list cannot be empty")

	// ErrEmptySubject indicates that email subject is missing
	ErrEmptySubject = errors.New("email subject cannot be empty")

	// ErrEmptyBody indicates that email body is missing
	ErrEmptyBody = errors.New("email body cannot be empty")
)