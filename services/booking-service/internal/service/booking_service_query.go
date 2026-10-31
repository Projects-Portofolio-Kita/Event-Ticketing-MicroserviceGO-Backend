// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_query.go
// Usability: Implements GetReservation to query reservation state and auto-expire overdue locked seats.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetReservation retrieves reservation details and updates status if expired.
func (s *BookingService) GetReservation(ctx context.Context, id uuid.UUID) (*dto.ReservationResponse, error) {
	reservation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}
	if reservation.Status == model.StatusLocked && time.Now().After(reservation.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, reservation.ID, model.StatusExpired)
		reservation.Status = model.StatusExpired
		s.releaseSeatLocks(ctx, reservation.EventID, reservation.Items)
	}
	return toReservationResponse(reservation), nil
}
