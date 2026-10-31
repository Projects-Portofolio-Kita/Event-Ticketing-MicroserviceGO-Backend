// Package service implements notification business logic and background queue dispatching.
//
// File: service_payment_failed.go
// Usability: Validates and enqueues payment failure and expiration notice notifications.
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	"github.com/google/uuid"
)

// SendPaymentFailed validates input, schedules the asynchronous job to render payment failure /
// reservation expiration templates, and sends the email via SMTP.
func (s *notificationService) SendPaymentFailed(ctx context.Context, req dto.PaymentFailedRequest) (string, error) {
	if err := s.validatePaymentFailed(req); err != nil { return "", err }
	jobID := uuid.New().String()
	job := queue.NewJob(jobID, func(jobCtx context.Context) error {
		renderResult, err := s.templateEngine.RenderPaymentFailed(template.PaymentFailedData{
			BookingRef: req.BookingRef, CustomerName: req.CustomerName, EventTitle: req.EventTitle,
			Reason: req.Reason, ExpiredAt: req.ExpiredAt,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to render payment failed template for booking %s: %v", req.BookingRef, err)
			return fmt.Errorf("failed to render payment failed template: %w", err)
		}
		msg := &smtp.EmailMessage{
			To: []string{req.CustomerEmail}, Subject: fmt.Sprintf("Payment Expired: Booking %s - %s", req.BookingRef, req.EventTitle),
			HTMLBody: renderResult.HTML, TextBody: renderResult.Text,
		}
		if err := s.emailSender.Send(jobCtx, msg); err != nil {
			log.Printf("[ERROR] Failed to dispatch payment failed email to %s for booking %s: %v", req.CustomerEmail, req.BookingRef, err)
			return fmt.Errorf("failed to send payment failed email: %w", err)
		}
		log.Printf("[INFO] Payment failed email successfully dispatched to %s (job %s)", req.CustomerEmail, jobID)
		return nil
	})
	if err := s.workerPool.Submit(job); err != nil {
		return "", fmt.Errorf("failed to submit payment failed job to worker pool: %w", err)
	}
	return jobID, nil
}