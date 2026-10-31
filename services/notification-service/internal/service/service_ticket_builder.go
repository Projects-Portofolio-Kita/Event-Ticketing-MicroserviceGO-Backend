// Package service implements notification business logic and background queue dispatching.
//
// File: service_ticket_builder.go
// Usability: Builds QR verification codes, seat items, and PDF models for ticket orders.
package service

import (
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/pdf"
)

func (s *notificationService) buildTicketPDFData(req dto.TicketConfirmationRequest) ([]pdf.TicketPDFData, []byte, error) {
	pdfTickets := make([]pdf.TicketPDFData, len(req.Seats))
	var firstQRBytes []byte
	for i, seat := range req.Seats {
		qrPayload := s.qrGen.GenerateVerificationPayload(req.BookingRef, seat.SeatID, req.EventID)
		qrBytes, err := s.qrGen.Generate(qrPayload)
		if err != nil {
			log.Printf("[ERROR] Failed to generate QR code for seat %s (booking %s): %v", seat.SeatID, req.BookingRef, err)
			return nil, nil, fmt.Errorf("failed to generate qr code: %w", err)
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
			BookingRef: req.BookingRef, EventTitle: req.EventTitle, EventDate: req.EventDate,
			VenueName: req.VenueName, VenueAddress: req.VenueAddress, AttendeeName: req.CustomerName,
			SeatDetails: seatDetail, TierName: req.TierName, TotalAmount: priceStr,
			Currency: req.Currency, QRData: qrPayload, QRBytes: qrBytes,
		}
	}
	return pdfTickets, firstQRBytes, nil
}