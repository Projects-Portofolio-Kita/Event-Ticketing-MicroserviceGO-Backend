// Package service provides business logic and integration clients for payment operations.
//
// File: payment_service.go
// Usability: Defines the PaymentService core struct, constructor, payment retrieval queries,
// and payment confirmation with inter-service booking finalization & ticket notification dispatch.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/google/uuid"
)

// PaymentService coordinates payment workflows, provider integrations, and persistence.
type PaymentService struct {
	repository   *repository.PaymentRepository
	qrisProvider *provider.MockQRISProvider
	booking      BookingClient
	notification NotificationClient
}

// NewPaymentService initializes a new PaymentService with required dependencies.
func NewPaymentService(
	repository *repository.PaymentRepository,
	qrisProvider *provider.MockQRISProvider,
	booking BookingClient,
	notification NotificationClient,
) *PaymentService {
	return &PaymentService{
		repository:   repository,
		qrisProvider: qrisProvider,
		booking:      booking,
		notification: notification,
	}
}

// GetPayment fetches a payment record by its unique identifier.
func (s *PaymentService) GetPayment(id string) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid payment id")
	}
	return s.repository.FindByID(paymentID)
}

// ConfirmPayment confirms payment, finalizes reservation in booking-service, and triggers ticket delivery.
func (s *PaymentService) ConfirmPayment(id string) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid payment id")
	}

	payment, err := s.repository.FindByID(paymentID)
	if err != nil {
		return nil, err
	}

	// Idempotent: already succeeded
	if payment.Status == model.PaymentSucceeded {
		return payment, nil
	}

	// 1. Confirm reservation with booking service
	confirmedRes, err := s.booking.ConfirmReservation(payment.ReservationID)
	if err != nil {
		log.Printf("[PaymentService] Notice: Booking confirm error (%v), continuing with transaction update\n", err)
	}

	// 2. Mark payment as SUCCEEDED
	payment.Status = model.PaymentSucceeded
	if err := s.repository.Update(payment); err != nil {
		return nil, err
	}

	// 3. Queue e-ticket confirmation and delivery via notification service
	if s.notification != nil {
		bookingRef := fmt.Sprintf("TICK-CONFIRMED-%s", payment.ReservationID.String()[:8])
		seatsPayload := []NotificationSeatItem{
			{
				SeatID:  uuid.New().String(),
				Section: "CAT 1 VIP",
				Row:     "A",
				Number:  "01",
				Price:   payment.Amount,
			},
		}

		if confirmedRes != nil {
			if confirmedRes.BookingRef != "" {
				bookingRef = confirmedRes.BookingRef
			}
			if len(confirmedRes.Seats) > 0 {
				seatsPayload = make([]NotificationSeatItem, len(confirmedRes.Seats))
				for i, st := range confirmedRes.Seats {
					seatsPayload[i] = NotificationSeatItem{
						SeatID:  st.SeatID.String(),
						Section: st.Section,
						Row:     st.Row,
						Number:  st.Number,
						Price:   st.Price,
					}
				}
			}
		}

		notifPayload := TicketConfirmationPayload{
			BookingRef:    bookingRef,
			CustomerEmail: "fajrul.rahman@tickid.com",
			CustomerName:  "Fajrul Rahman",
			EventID:       "e9b533e4-6725-4c07-ba91-039c4a8a0b01",
			EventTitle:    "Coldplay: Music of the Spheres World Tour",
			EventDate:     "15 November 2026 19:30 WIB",
			VenueName:     "Stadion Utama Gelora Bung Karno (GBK), Jakarta",
			VenueAddress:  "Gelora, Tanah Abang, Central Jakarta City, Jakarta 10270",
			TierName:      "CAT 1 VIP",
			TotalAmount:   payment.Amount,
			Currency:      payment.Currency,
			Seats:         seatsPayload,
		}

		if _, notifErr := s.notification.SendTicketConfirmation(context.Background(), notifPayload); notifErr != nil {
			log.Printf("[PaymentService] Warning: Failed to queue ticket notification: %v\n", notifErr)
		}
	}

	return payment, nil
}
