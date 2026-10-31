// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_mapper.go
// Usability: Provides helper functions to transform Reservation entities into DTOs and generate reference codes.
package service

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
)

func toReservationResponse(r *model.Reservation) *dto.ReservationResponse {
	seats := make([]dto.SeatResponseDTO, len(r.Items))
	for i, item := range r.Items {
		seats[i] = dto.SeatResponseDTO{
			SeatID: item.SeatID, Section: item.Section, Row: item.Row, Number: item.Number, Price: item.Price,
		}
	}
	return &dto.ReservationResponse{
		ReservationID: r.ID,
		UserID:        r.UserID,
		EventID:       r.EventID,
		BookingRef:    r.BookingRef,
		Status:        string(r.Status),
		ExpiresAt:     r.ExpiresAt,
		TotalAmount:   r.TotalAmount,
		Currency:      r.Currency,
		Seats:         seats,
		CreatedAt:     r.CreatedAt,
	}
}

func generateBookingRef() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil { return "", err }
		b[i] = charset[num.Int64()]
	}
	return fmt.Sprintf("TKT-2026-%s", string(b)), nil
}