// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_lock.go
// Usability: Handles multi-seat atomic distributed lock acquisition and rollback operations.
package service

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
)

func (s *BookingService) acquireSeatLocks(ctx context.Context, eventIDStr, userIDStr string, seats []dto.SeatItemDTO) ([]string, error) {
	var acquiredSeats []string
	for _, seat := range seats {
		seatIDStr := seat.SeatID.String()
		locked, lockErr := s.locker.LockSeat(ctx, eventIDStr, seatIDStr, userIDStr, s.ttl)
		if lockErr != nil || !locked {
			_ = s.locker.ReleaseSeats(ctx, eventIDStr, acquiredSeats)
			return nil, ErrSeatsUnavailable
		}
		acquiredSeats = append(acquiredSeats, seatIDStr)
	}
	return acquiredSeats, nil
}

func (s *BookingService) releaseSeatLocks(ctx context.Context, eventID uuid.UUID, items []model.ReservationItem) {
	for _, item := range items {
		_ = s.locker.ReleaseSeat(ctx, eventID.String(), item.SeatID.String())
	}
}
