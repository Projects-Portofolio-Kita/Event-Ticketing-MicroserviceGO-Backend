// Package service implements notification business logic and background queue dispatching.
//
// File: service_custom_email.go
// Usability: Validates and enqueues custom email dispatching through the background worker pool.
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	"github.com/google/uuid"
)

// SendCustomEmail validates input and enqueues custom email dispatch.
func (s *notificationService) SendCustomEmail(ctx context.Context, req dto.CustomEmailRequest) (string, error) {
	if err := s.validateCustomEmail(req); err != nil { return "", err }
	jobID := uuid.New().String()
	job := queue.NewJob(jobID, func(jobCtx context.Context) error {
		htmlBody, textBody := "", req.Body
		if req.IsHTML { htmlBody = req.Body }
		msg := &smtp.EmailMessage{To: req.To, Subject: req.Subject, HTMLBody: htmlBody, TextBody: textBody}
		if err := s.emailSender.Send(jobCtx, msg); err != nil {
			log.Printf("[ERROR] Failed to dispatch custom email to %v (job %s): %v", req.To, jobID, err)
			return fmt.Errorf("failed to send custom email: %w", err)
		}
		log.Printf("[INFO] Custom email successfully dispatched to %v (job %s)", req.To, jobID)
		return nil
	})
	if err := s.workerPool.Submit(job); err != nil {
		return "", fmt.Errorf("failed to submit custom email job to worker pool: %w", err)
	}
	return jobID, nil
}