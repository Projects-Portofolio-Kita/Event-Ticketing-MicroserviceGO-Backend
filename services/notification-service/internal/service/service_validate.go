// Package service implements notification business logic and background queue dispatching.
//
// File: service_validate.go
// Usability: Validates input requests for ticket confirmations, payment failures, and custom emails.
package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
)

func (s *notificationService) validateTicketConfirmation(req dto.TicketConfirmationRequest) error {
	if strings.TrimSpace(req.BookingRef) == "" { return ErrEmptyBookingRef }
	if !isValidEmail(req.CustomerEmail) { return ErrInvalidEmail }
	if strings.TrimSpace(req.CustomerName) == "" { return ErrEmptyCustomerName }
	if strings.TrimSpace(req.EventTitle) == "" { return ErrEmptyEventTitle }
	if len(req.Seats) == 0 { return ErrEmptySeats }
	for _, seat := range req.Seats {
		if strings.TrimSpace(seat.SeatID) == "" || strings.TrimSpace(seat.Number) == "" {
			return errors.New("seat_id and number are required for every seat")
		}
	}
	return nil
}

func (s *notificationService) validatePaymentFailed(req dto.PaymentFailedRequest) error {
	if strings.TrimSpace(req.BookingRef) == "" { return ErrEmptyBookingRef }
	if !isValidEmail(req.CustomerEmail) { return ErrInvalidEmail }
	if strings.TrimSpace(req.CustomerName) == "" { return ErrEmptyCustomerName }
	if strings.TrimSpace(req.EventTitle) == "" { return ErrEmptyEventTitle }
	return nil
}

func (s *notificationService) validateCustomEmail(req dto.CustomEmailRequest) error {
	if len(req.To) == 0 { return ErrEmptyRecipients }
	for _, recipient := range req.To {
		if !isValidEmail(recipient) { return fmt.Errorf("%w: %s", ErrInvalidEmail, recipient) }
	}
	if strings.TrimSpace(req.Subject) == "" { return ErrEmptySubject }
	if strings.TrimSpace(req.Body) == "" { return ErrEmptyBody }
	return nil
}