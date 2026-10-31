// Package service implements notification business logic and background queue dispatching.
//
// File: service_ticket.go
// Usability: Validates and queues asynchronous ticket confirmation notification tasks.
package service

import (
	"context"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/google/uuid"
)

// SendTicketConfirmation validates input, schedules the asynchronous job to generate QR codes,
// render E-Ticket PDF, compile responsive HTML/text emails, and deliver via SMTP.
func (s *notificationService) SendTicketConfirmation(ctx context.Context, req dto.TicketConfirmationRequest) (string, error) {
	if err := s.validateTicketConfirmation(req); err != nil {
		return "", err
	}
	jobID := uuid.New().String()
	job := queue.NewJob(jobID, func(jobCtx context.Context) error {
		return s.processTicketConfirmation(jobCtx, jobID, req)
	})
	if err := s.workerPool.Submit(job); err != nil {
		return "", fmt.Errorf("failed to submit ticket confirmation job to worker pool: %w", err)
	}
	return jobID, nil
}