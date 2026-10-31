// Package service implements notification business logic and background queue dispatching.
//
// File: service_ticket_render.go
// Usability: Populates order confirmation template structures and invokes the HTML/text engine.
package service

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
)

func (s *notificationService) renderOrderConfirmation(req dto.TicketConfirmationRequest, firstQR []byte) (*template.RenderResult, error) {
	templateSeats := make([]template.SeatItem, len(req.Seats))
	for i, st := range req.Seats {
		templateSeats[i] = template.SeatItem{
			SeatID: st.SeatID, Section: st.Section, Row: st.Row, Number: st.Number, PriceFormatted: formatAmount(st.Price),
		}
	}
	qrBase64 := ""
	if len(firstQR) > 0 {
		qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(firstQR)
	}
	currency := req.Currency
	if currency == "" { currency = "IDR" }
	res, err := s.templateEngine.RenderOrderConfirmation(template.OrderConfirmationData{
		BookingRef: req.BookingRef, CustomerName: req.CustomerName, CustomerEmail: req.CustomerEmail,
		EventTitle: req.EventTitle, EventDate: req.EventDate, VenueName: req.VenueName,
		VenueAddress: req.VenueAddress, TierName: req.TierName, Seats: templateSeats,
		TotalAmountFormatted: formatAmount(req.TotalAmount), Currency: currency, QRCodeBase64: qrBase64, HasPDFAttachment: true,
	})
	if err != nil {
		log.Printf("[ERROR] Failed to render order confirmation template for booking %s: %v", req.BookingRef, err)
		return nil, fmt.Errorf("failed to render confirmation template: %w", err)
	}
	return res, nil
}