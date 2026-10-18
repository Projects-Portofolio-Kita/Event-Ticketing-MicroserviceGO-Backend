package service_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/pdf"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	rscpdf "rsc.io/pdf"
)

// In-memory spy email sender for capturing dispatched emails and inspecting payloads
type spyEmailSender struct {
	mu       sync.Mutex
	messages []*smtp.EmailMessage
}

func (s *spyEmailSender) Send(ctx context.Context, msg *smtp.EmailMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
	return nil
}

func (s *spyEmailSender) LastMessage() *smtp.EmailMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) == 0 {
		return nil
	}
	return s.messages[len(s.messages)-1]
}

func (s *spyEmailSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

// 1. Multi-Seat End-to-End Pipeline (10 to 20 seats)
func TestAdversarial_Service_MultiSeatPipeline(t *testing.T) {
	spySender := &spyEmailSender{}
	workerPool := queue.NewWorkerPool(4, 50)
	workerPool.Start(context.Background())
	defer func() {
		_ = workerPool.Stop(context.Background())
	}()

	qrGen := pdf.NewQRCodeGenerator(256)
	pdfGen := pdf.NewTicketPDFGenerator(qrGen)
	tmplEngine, err := template.NewEngine()
	if err != nil {
		t.Fatalf("failed to create template engine: %v", err)
	}

	svc, err := service.NewNotificationService(spySender, workerPool, pdfGen, qrGen, tmplEngine)
	if err != nil {
		t.Fatalf("failed to create notification service: %v", err)
	}

	seatCounts := []int{10, 20}
	for _, count := range seatCounts {
		t.Run(fmt.Sprintf("%d_Seats_Order", count), func(t *testing.T) {
			seats := make([]dto.SeatInfoDTO, count)
			for i := 0; i < count; i++ {
				seats[i] = dto.SeatInfoDTO{
					SeatID:  fmt.Sprintf("SEAT-UUID-%04d", i+1),
					Section: fmt.Sprintf("Zone-%c", 'A'+(i%4)),
					Row:     fmt.Sprintf("R%02d", (i/4)+1),
					Number:  fmt.Sprintf("S%03d", i+1),
					Price:   250000,
				}
			}

			bookingRef := fmt.Sprintf("BOOK-PIPELINE-%d", count)
			req := dto.TicketConfirmationRequest{
				BookingRef:    bookingRef,
				CustomerEmail: fmt.Sprintf("group%d@example.com", count),
				CustomerName:  "Festival Group Organizer",
				EventID:       "EVT-MEGA-CONCERT",
				EventTitle:    "Mega Stadium Festival 2026",
				EventDate:     "Saturday, 20 Dec 2026, 18:00 WIB",
				VenueName:     "GBK Senayan Stadium",
				VenueAddress:  "Jl. Pintu Satu Senayan, Jakarta",
				TierName:      "Early Bird Festival",
				TotalAmount:   float64(count * 250000),
				Currency:      "IDR",
				Seats:         seats,
			}

			initialCount := spySender.Count()
			jobID, err := svc.SendTicketConfirmation(context.Background(), req)
			if err != nil {
				t.Fatalf("failed to dispatch confirmation: %v", err)
			}
			if jobID == "" {
				t.Fatal("expected non-empty job ID")
			}

			// Wait for async worker execution
			deadline := time.Now().Add(5 * time.Second)
			for spySender.Count() <= initialCount && time.Now().Before(deadline) {
				time.Sleep(20 * time.Millisecond)
			}

			if spySender.Count() <= initialCount {
				t.Fatalf("worker did not dispatch email within deadline for %d seats", count)
			}

			msg := spySender.LastMessage()
			if msg == nil {
				t.Fatal("captured email message is nil")
			}

			// Verify recipient and subject
			if len(msg.To) != 1 || msg.To[0] != req.CustomerEmail {
				t.Fatalf("expected recipient %s, got %v", req.CustomerEmail, msg.To)
			}
			expectedSubject := fmt.Sprintf("Your Tickets for %s - Booking Ref %s", req.EventTitle, req.BookingRef)
			if msg.Subject != expectedSubject {
				t.Fatalf("subject mismatch.\nExpected: %s\nGot:      %s", expectedSubject, msg.Subject)
			}

			// Verify PDF Attachment
			if len(msg.Attachments) != 1 {
				t.Fatalf("expected exactly 1 PDF attachment, got %d", len(msg.Attachments))
			}
			att := msg.Attachments[0]
			expectedFilename := fmt.Sprintf("ticket_%s.pdf", bookingRef)
			if att.Filename != expectedFilename {
				t.Errorf("attachment filename mismatch: expected %s, got %s", expectedFilename, att.Filename)
			}
			if att.ContentType != "application/pdf" {
				t.Errorf("attachment content type mismatch: expected application/pdf, got %s", att.ContentType)
			}

			// Verify PDF structure and page count
			pdfReader, err := rscpdf.NewReader(bytes.NewReader(att.Data), int64(len(att.Data)))
			if err != nil {
				t.Fatalf("failed to parse PDF attachment with rsc.io/pdf: %v", err)
			}
			if pdfReader.NumPage() != count {
				t.Fatalf("PDF attachment page count mismatch: expected %d pages, got %d pages", count, pdfReader.NumPage())
			}

			// Verify HTML seat list contains all seats
			for i := 0; i < count; i++ {
				seatNum := fmt.Sprintf("S%03d", i+1)
				if !strings.Contains(msg.HTMLBody, seatNum) {
					t.Errorf("HTML email missing seat %s", seatNum)
				}
			}
		})
	}
}

// 2. 0 IDR Free Tickets & Billions IDR Boundary Pipeline
func TestAdversarial_Service_PriceBoundaries(t *testing.T) {
	spySender := &spyEmailSender{}
	workerPool := queue.NewWorkerPool(2, 20)
	workerPool.Start(context.Background())
	defer func() {
		_ = workerPool.Stop(context.Background())
	}()

	svc, err := service.NewNotificationService(spySender, workerPool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialize service: %v", err)
	}

	testCases := []struct {
		name          string
		totalAmount   float64
		seatPrice     float64
		expectedTotal string
	}{
		{
			name:          "0 IDR Free RSVP Ticket",
			totalAmount:   0,
			seatPrice:     0,
			expectedTotal: "0 IDR",
		},
		{
			name:          "Auto-split Total Amount when seat price is zero",
			totalAmount:   1000000,
			seatPrice:     0,
			expectedTotal: "1,000,000 IDR",
		},
		{
			name:          "1 Billion IDR Ultra VIP",
			totalAmount:   1000000000,
			seatPrice:     1000000000,
			expectedTotal: "1,000,000,000 IDR",
		},
		{
			name:          "500 Billion IDR Extreme Boundary",
			totalAmount:   500000000000,
			seatPrice:     500000000000,
			expectedTotal: "500,000,000,000 IDR",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bookingRef := fmt.Sprintf("BOOK-PRICE-%d", int64(tc.totalAmount))
			req := dto.TicketConfirmationRequest{
				BookingRef:    bookingRef,
				CustomerEmail: "buyer@example.com",
				CustomerName:  "Test Buyer",
				EventID:       "EVT-PRICE",
				EventTitle:    "Price Test Gala",
				EventDate:     "Sunday, 1 Nov 2026",
				VenueName:     "Grand Ballroom",
				TierName:      "Standard",
				TotalAmount:   tc.totalAmount,
				Currency:      "IDR",
				Seats: []dto.SeatInfoDTO{
					{
						SeatID: "S-1",
						Number: "A-01",
						Price:  tc.seatPrice,
					},
				},
			}

			initialCount := spySender.Count()
			_, err := svc.SendTicketConfirmation(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error dispatching ticket confirmation: %v", err)
			}

			// Wait for worker
			deadline := time.Now().Add(3 * time.Second)
			for spySender.Count() <= initialCount && time.Now().Before(deadline) {
				time.Sleep(20 * time.Millisecond)
			}

			if spySender.Count() <= initialCount {
				t.Fatalf("worker timed out processing price case %s", tc.name)
			}

			msg := spySender.LastMessage()
			if !strings.Contains(msg.HTMLBody, tc.expectedTotal) {
				t.Errorf("HTML body missing expected price %q", tc.expectedTotal)
			}
			if !strings.Contains(msg.TextBody, tc.expectedTotal) {
				t.Errorf("Text body missing expected price %q", tc.expectedTotal)
			}
		})
	}
}

// 3. Email Header Injection and Malformed Input Protection
func TestAdversarial_Service_EmailHeaderInjection(t *testing.T) {
	spySender := &spyEmailSender{}
	workerPool := queue.NewWorkerPool(1, 10)
	workerPool.Start(context.Background())
	defer func() {
		_ = workerPool.Stop(context.Background())
	}()

	svc, err := service.NewNotificationService(spySender, workerPool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialize service: %v", err)
	}

	maliciousEmails := []string{
		"victim@example.com\r\nBcc: hacker@example.com",
		"victim@example.com\nBcc: hacker@example.com",
		"victim@example.com\r\nSubject: Injected Subject",
		"attacker@evil\x00.com",
		"\"hacker\"@example.com\r\nTo: hijacked@example.com",
		"not-an-email",
		"missingdomain@",
		"@missinguser.com",
		"spaces in email@example.com",
	}

	for _, badEmail := range maliciousEmails {
		t.Run("Reject_"+badEmail, func(t *testing.T) {
			req := dto.TicketConfirmationRequest{
				BookingRef:    "REF-HDR-INJ",
				CustomerEmail: badEmail,
				CustomerName:  "Test",
				EventID:       "E-1",
				EventTitle:    "Test",
				EventDate:     "Today",
				VenueName:     "Venue",
				TierName:      "General",
				TotalAmount:   100000,
				Seats: []dto.SeatInfoDTO{
					{SeatID: "S1", Number: "1"},
				},
			}

			_, err := svc.SendTicketConfirmation(context.Background(), req)
			if err == nil {
				t.Errorf("expected validation error for malicious/invalid email %q, got nil", badEmail)
			}
		})
	}
}
