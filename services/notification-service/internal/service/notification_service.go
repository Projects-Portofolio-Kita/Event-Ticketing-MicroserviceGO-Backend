package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/pdf"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	"github.com/google/uuid"
)

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

// NotificationService defines the business operations for asynchronous email notifications.
type NotificationService interface {
	SendTicketConfirmation(ctx context.Context, req dto.TicketConfirmationRequest) (string, error)
	SendPaymentFailed(ctx context.Context, req dto.PaymentFailedRequest) (string, error)
	SendCustomEmail(ctx context.Context, req dto.CustomEmailRequest) (string, error)
}

// notificationService is the concrete implementation of NotificationService.
type notificationService struct {
	emailSender    smtp.EmailSender
	workerPool     queue.WorkerPool
	pdfGen         pdf.PDFGenerator
	qrGen          pdf.QRCodeGenerator
	templateEngine template.Engine
}

// NewNotificationService creates an initialized NotificationService.
// If pdfGen, qrGen, or templateEngine are nil, default implementations are created.
func NewNotificationService(
	emailSender smtp.EmailSender,
	workerPool queue.WorkerPool,
	pdfGen pdf.PDFGenerator,
	qrGen pdf.QRCodeGenerator,
	templateEngine template.Engine,
) (NotificationService, error) {
	if emailSender == nil {
		return nil, errors.New("emailSender cannot be nil")
	}
	if workerPool == nil {
		return nil, errors.New("workerPool cannot be nil")
	}

	if qrGen == nil {
		qrGen = pdf.NewQRCodeGenerator(pdf.DefaultQRSize)
	}
	if pdfGen == nil {
		pdfGen = pdf.NewTicketPDFGenerator(qrGen)
	}
	if templateEngine == nil {
		var err error
		templateEngine, err = template.NewEngine()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize template engine: %w", err)
		}
	}

	return &notificationService{
		emailSender:    emailSender,
		workerPool:     workerPool,
		pdfGen:         pdfGen,
		qrGen:          qrGen,
		templateEngine: templateEngine,
	}, nil
}

// SendTicketConfirmation validates input, schedules the asynchronous job to generate QR codes,
// render E-Ticket PDF, compile responsive HTML/text emails, and deliver via SMTP.
func (s *notificationService) SendTicketConfirmation(ctx context.Context, req dto.TicketConfirmationRequest) (string, error) {
	if err := s.validateTicketConfirmation(req); err != nil {
		return "", err
	}

	jobID := uuid.New().String()

	job := queue.NewJob(jobID, func(jobCtx context.Context) error {
		// 1. Generate QR payloads and PDF data per seat
		pdfTickets := make([]pdf.TicketPDFData, len(req.Seats))
		var firstQRBytes []byte

		for i, seat := range req.Seats {
			qrPayload := s.qrGen.GenerateVerificationPayload(req.BookingRef, seat.SeatID, req.EventID)
			qrBytes, err := s.qrGen.Generate(qrPayload)
			if err != nil {
				log.Printf("[ERROR] Failed to generate QR code for seat %s (booking %s): %v", seat.SeatID, req.BookingRef, err)
				return fmt.Errorf("failed to generate qr code: %w", err)
			}
			if i == 0 {
				firstQRBytes = qrBytes
			}

			seatDetail := seat.Number
			if seat.Section != "" && seat.Row != "" {
				seatDetail = fmt.Sprintf("Sec %s / Row %s / Seat %s", seat.Section, seat.Row, seat.Number)
			} else if seat.Section != "" {
				seatDetail = fmt.Sprintf("Sec %s / Seat %s", seat.Section, seat.Number)
			}

			priceStr := formatAmount(seat.Price)
			if seat.Price == 0 && req.TotalAmount > 0 && len(req.Seats) > 0 {
				priceStr = formatAmount(req.TotalAmount / float64(len(req.Seats)))
			}

			pdfTickets[i] = pdf.TicketPDFData{
				BookingRef:   req.BookingRef,
				EventTitle:   req.EventTitle,
				EventDate:    req.EventDate,
				VenueName:    req.VenueName,
				VenueAddress: req.VenueAddress,
				AttendeeName: req.CustomerName,
				SeatDetails:  seatDetail,
				TierName:     req.TierName,
				TotalAmount:  priceStr,
				Currency:     req.Currency,
				QRData:       qrPayload,
				QRBytes:      qrBytes,
			}
		}

		// 2. Generate Multi-ticket PDF document
		pdfBytes, err := s.pdfGen.GenerateMultiTicket(pdfTickets)
		if err != nil {
			log.Printf("[ERROR] Failed to generate e-ticket PDF for booking %s: %v", req.BookingRef, err)
			return fmt.Errorf("failed to generate pdf: %w", err)
		}

		// 3. Render HTML and Text order confirmation templates
		templateSeats := make([]template.SeatItem, len(req.Seats))
		for i, st := range req.Seats {
			templateSeats[i] = template.SeatItem{
				SeatID:         st.SeatID,
				Section:        st.Section,
				Row:            st.Row,
				Number:         st.Number,
				PriceFormatted: formatAmount(st.Price),
			}
		}

		qrBase64 := ""
		if len(firstQRBytes) > 0 {
			qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(firstQRBytes)
		}

		currency := req.Currency
		if currency == "" {
			currency = "IDR"
		}

		renderResult, err := s.templateEngine.RenderOrderConfirmation(template.OrderConfirmationData{
			BookingRef:           req.BookingRef,
			CustomerName:         req.CustomerName,
			CustomerEmail:        req.CustomerEmail,
			EventTitle:           req.EventTitle,
			EventDate:            req.EventDate,
			VenueName:            req.VenueName,
			VenueAddress:         req.VenueAddress,
			TierName:             req.TierName,
			Seats:                templateSeats,
			TotalAmountFormatted: formatAmount(req.TotalAmount),
			Currency:             currency,
			QRCodeBase64:         qrBase64,
			HasPDFAttachment:     true,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to render order confirmation template for booking %s: %v", req.BookingRef, err)
			return fmt.Errorf("failed to render confirmation template: %w", err)
		}

		// 4. Build and send email with PDF attachment
		msg := &smtp.EmailMessage{
			To:       []string{req.CustomerEmail},
			Subject:  fmt.Sprintf("Your Tickets for %s - Booking Ref %s", req.EventTitle, req.BookingRef),
			HTMLBody: renderResult.HTML,
			TextBody: renderResult.Text,
			Attachments: []smtp.Attachment{
				{
					Filename:    fmt.Sprintf("ticket_%s.pdf", req.BookingRef),
					ContentType: "application/pdf",
					Data:        pdfBytes,
				},
			},
		}

		if err := s.emailSender.Send(jobCtx, msg); err != nil {
			log.Printf("[ERROR] Failed to dispatch ticket confirmation email to %s for booking %s: %v", req.CustomerEmail, req.BookingRef, err)
			return fmt.Errorf("failed to send ticket confirmation email: %w", err)
		}

		log.Printf("[INFO] Ticket confirmation email successfully dispatched to %s (job %s)", req.CustomerEmail, jobID)
		return nil
	})

	if err := s.workerPool.Submit(job); err != nil {
		return "", fmt.Errorf("failed to submit ticket confirmation job to worker pool: %w", err)
	}

	return jobID, nil
}

// SendPaymentFailed validates input, schedules the asynchronous job to render payment failure /
// reservation expiration templates, and sends the email via SMTP.
func (s *notificationService) SendPaymentFailed(ctx context.Context, req dto.PaymentFailedRequest) (string, error) {
	if err := s.validatePaymentFailed(req); err != nil {
		return "", err
	}

	jobID := uuid.New().String()

	job := queue.NewJob(jobID, func(jobCtx context.Context) error {
		renderResult, err := s.templateEngine.RenderPaymentFailed(template.PaymentFailedData{
			BookingRef:   req.BookingRef,
			CustomerName: req.CustomerName,
			EventTitle:   req.EventTitle,
			Reason:       req.Reason,
			ExpiredAt:    req.ExpiredAt,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to render payment failed template for booking %s: %v", req.BookingRef, err)
			return fmt.Errorf("failed to render payment failed template: %w", err)
		}

		msg := &smtp.EmailMessage{
			To:       []string{req.CustomerEmail},
			Subject:  fmt.Sprintf("Payment Expired: Booking %s - %s", req.BookingRef, req.EventTitle),
			HTMLBody: renderResult.HTML,
			TextBody: renderResult.Text,
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

// SendCustomEmail validates input and enqueues custom email dispatch.
func (s *notificationService) SendCustomEmail(ctx context.Context, req dto.CustomEmailRequest) (string, error) {
	if err := s.validateCustomEmail(req); err != nil {
		return "", err
	}

	jobID := uuid.New().String()

	job := queue.NewJob(jobID, func(jobCtx context.Context) error {
		var htmlBody, textBody string
		if req.IsHTML {
			htmlBody = req.Body
			textBody = req.Body
		} else {
			textBody = req.Body
		}

		msg := &smtp.EmailMessage{
			To:       req.To,
			Subject:  req.Subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}

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

// validateTicketConfirmation checks required fields for TicketConfirmationRequest.
func (s *notificationService) validateTicketConfirmation(req dto.TicketConfirmationRequest) error {
	if strings.TrimSpace(req.BookingRef) == "" {
		return ErrEmptyBookingRef
	}
	if !isValidEmail(req.CustomerEmail) {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		return ErrEmptyCustomerName
	}
	if strings.TrimSpace(req.EventTitle) == "" {
		return ErrEmptyEventTitle
	}
	if len(req.Seats) == 0 {
		return ErrEmptySeats
	}
	for _, seat := range req.Seats {
		if strings.TrimSpace(seat.SeatID) == "" || strings.TrimSpace(seat.Number) == "" {
			return errors.New("seat_id and number are required for every seat")
		}
	}
	return nil
}

// validatePaymentFailed checks required fields for PaymentFailedRequest.
func (s *notificationService) validatePaymentFailed(req dto.PaymentFailedRequest) error {
	if strings.TrimSpace(req.BookingRef) == "" {
		return ErrEmptyBookingRef
	}
	if !isValidEmail(req.CustomerEmail) {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		return ErrEmptyCustomerName
	}
	if strings.TrimSpace(req.EventTitle) == "" {
		return ErrEmptyEventTitle
	}
	return nil
}

// validateCustomEmail checks required fields for CustomEmailRequest.
func (s *notificationService) validateCustomEmail(req dto.CustomEmailRequest) error {
	if len(req.To) == 0 {
		return ErrEmptyRecipients
	}
	for _, recipient := range req.To {
		if !isValidEmail(recipient) {
			return fmt.Errorf("%w: %s", ErrInvalidEmail, recipient)
		}
	}
	if strings.TrimSpace(req.Subject) == "" {
		return ErrEmptySubject
	}
	if strings.TrimSpace(req.Body) == "" {
		return ErrEmptyBody
	}
	return nil
}

// isValidEmail validates standard email address formatting.
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	return strings.Contains(parts[1], ".")
}

// formatAmount formats numbers with comma thousands separators or two decimal places.
func formatAmount(amount float64) string {
	if amount != float64(int64(amount)) {
		return fmt.Sprintf("%.2f", amount)
	}
	val := int64(amount)
	if val < 0 {
		return fmt.Sprintf("%d", val)
	}
	s := fmt.Sprintf("%d", val)
	if len(s) <= 3 {
		return s
	}
	var res []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			res = append(res, ',')
		}
		res = append(res, byte(c))
	}
	return string(res)
}
