// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_cancel.go
// Usability: Implements CancelReservation and SweepExpired to cancel bookings and clean up expired holds.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CancelReservation releases the held seats and marks reservation CANCELLED.
func (s *BookingService) CancelReservation(ctx context.Context, id uuid.UUID) error {
	res, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReservationNotFound
		}
		return err
	}
	if res.Status != model.StatusLocked {
		return ErrInvalidStatusChange
	}
	if err := s.repo.UpdateStatus(ctx, res.ID, model.StatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel reservation: %w", err)
	}
	s.releaseSeatLocks(ctx, res.EventID, res.Items)
	return nil
}

// SweepExpired releases seats from expired reservations in batches.
func (s *BookingService) SweepExpired(ctx context.Context) (int, error) {
	expiredList, err := s.repo.FindExpiredLockedReservations(ctx, 100)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, res := range expiredList {
		if err := s.repo.UpdateStatus(ctx, res.ID, model.StatusExpired); err == nil {
			count++
			s.releaseSeatLocks(ctx, res.EventID, res.Items)
		}
	}
	return count, nil
}
