// Package service implements notification business logic and background queue dispatching.
//
// File: service_ticket_job.go
// Usability: Executes PDF rendering and email dispatch for confirmed ticket bookings.
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
)

func (s *notificationService) processTicketConfirmation(jobCtx context.Context, jobID string, req dto.TicketConfirmationRequest) error {
	pdfTickets, firstQR, err := s.buildTicketPDFData(req)
	if err != nil { return err }
	pdfBytes, err := s.pdfGen.GenerateMultiTicket(pdfTickets)
	if err != nil {
		log.Printf("[ERROR] Failed to generate e-ticket PDF for booking %s: %v", req.BookingRef, err)
		return fmt.Errorf("failed to generate pdf: %w", err)
	}
	renderResult, err := s.renderOrderConfirmation(req, firstQR)
	if err != nil { return err }
	msg := &smtp.EmailMessage{
		To:       []string{req.CustomerEmail},
		Subject:  fmt.Sprintf("Your Tickets for %s - Booking Ref %s", req.EventTitle, req.BookingRef),
		HTMLBody: renderResult.HTML,
		TextBody: renderResult.Text,
		Attachments: []smtp.Attachment{
			{Filename: fmt.Sprintf("ticket_%s.pdf", req.BookingRef), ContentType: "application/pdf", Data: pdfBytes},
		},
	}
	if err := s.emailSender.Send(jobCtx, msg); err != nil {
		log.Printf("[ERROR] Failed to dispatch ticket confirmation email to %s for booking %s: %v", req.CustomerEmail, req.BookingRef, err)
		return fmt.Errorf("failed to send ticket confirmation email: %w", err)
	}
	log.Printf("[INFO] Ticket confirmation email successfully dispatched to %s (job %s)", req.CustomerEmail, jobID)
	return nil
}