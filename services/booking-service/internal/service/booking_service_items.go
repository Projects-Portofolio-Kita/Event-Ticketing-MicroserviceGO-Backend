// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_items.go
// Usability: Provides helper function to construct ReservationItem entities from seat DTOs.
package service

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
)

func buildReservationItems(resID uuid.UUID, seats []dto.SeatItemDTO) ([]model.ReservationItem, float64) {
	var total float64
	items := make([]model.ReservationItem, len(seats))
	for i, seat := range seats {
		total += seat.Price
		items[i] = model.ReservationItem{
			ID:            uuid.New(),
			ReservationID: resID,
			SeatID:        seat.SeatID,
			Section:       seat.Section,
			Row:           seat.Row,
			Number:        seat.Number,
			Price:         seat.Price,
		}
	}
	return items, total
}
