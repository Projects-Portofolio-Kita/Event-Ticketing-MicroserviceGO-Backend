// Package service implements notification business logic and background queue dispatching.
//
// File: service_types.go
// Usability: Defines NotificationService contract, implementation struct, and factory constructor.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/pdf"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
)

// NotificationService defines the business operations for asynchronous email notifications.
type NotificationService interface {
	SendTicketConfirmation(ctx context.Context, req dto.TicketConfirmationRequest) (string, error)
	SendPaymentFailed(ctx context.Context, req dto.PaymentFailedRequest) (string, error)
	SendCustomEmail(ctx context.Context, req dto.CustomEmailRequest) (string, error)
}

type notificationService struct {
	emailSender    smtp.EmailSender
	workerPool     queue.WorkerPool
	pdfGen         pdf.PDFGenerator
	qrGen          pdf.QRCodeGenerator
	templateEngine template.Engine
}

// NewNotificationService creates an initialized NotificationService.
func NewNotificationService(emailSender smtp.EmailSender, workerPool queue.WorkerPool, pdfGen pdf.PDFGenerator, qrGen pdf.QRCodeGenerator, templateEngine template.Engine) (NotificationService, error) {
	if emailSender == nil { return nil, errors.New("emailSender cannot be nil") }
	if workerPool == nil { return nil, errors.New("workerPool cannot be nil") }
	if qrGen == nil { qrGen = pdf.NewQRCodeGenerator(pdf.DefaultQRSize) }
	if pdfGen == nil { pdfGen = pdf.NewTicketPDFGenerator(qrGen) }
	if templateEngine == nil {
		var err error
		templateEngine, err = template.NewEngine()
		if err != nil { return nil, fmt.Errorf("failed to initialize template engine: %w", err) }
	}
	return &notificationService{emailSender: emailSender, workerPool: workerPool, pdfGen: pdfGen, qrGen: qrGen, templateEngine: templateEngine}, nil
}