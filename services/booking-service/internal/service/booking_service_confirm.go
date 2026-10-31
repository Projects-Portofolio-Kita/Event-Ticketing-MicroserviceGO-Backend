// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_confirm.go
// Usability: Implements ConfirmReservation to mark payment confirmed and release temporary redis locks.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConfirmReservation marks a reservation as CONFIRMED upon successful payment.
func (s *BookingService) ConfirmReservation(ctx context.Context, id uuid.UUID) (*dto.ReservationResponse, error) {
	reservation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}
	if reservation.Status != model.StatusLocked {
		if reservation.Status == model.StatusConfirmed {
			return toReservationResponse(reservation), nil
		}
		return nil, ErrInvalidStatusChange
	}
	if time.Now().After(reservation.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, reservation.ID, model.StatusExpired)
		return nil, ErrReservationExpired
	}
	if err := s.repo.UpdateStatus(ctx, reservation.ID, model.StatusConfirmed); err != nil {
		return nil, fmt.Errorf("failed to confirm reservation: %w", err)
	}
	reservation.Status = model.StatusConfirmed
	s.releaseSeatLocks(ctx, reservation.EventID, reservation.Items)
	return toReservationResponse(reservation), nil
}
