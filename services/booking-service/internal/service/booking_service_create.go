// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service_create.go
// Usability: Implements CreateReservation with seat availability checking, Redis locking, and persistence.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
)

// CreateReservation executes atomic multi-seat distributed locking and persists the reservation.
func (s *BookingService) CreateReservation(ctx context.Context, req dto.CreateReservationRequest) (*dto.ReservationResponse, error) {
	if len(req.Seats) == 0 {
		return nil, errors.New("at least one seat must be selected")
	}
	seatUUIDs := make([]uuid.UUID, len(req.Seats))
	for i, seat := range req.Seats {
		seatUUIDs[i] = seat.SeatID
	}
	if booked, err := s.repo.CheckSeatsAlreadyBooked(ctx, req.EventID, seatUUIDs); err != nil {
		return nil, fmt.Errorf("failed to verify seat availability: %w", err)
	} else if booked {
		return nil, ErrSeatsUnavailable
	}
	eID, uID := req.EventID.String(), req.UserID.String()
	acquired, err := s.acquireSeatLocks(ctx, eID, uID, req.Seats)
	if err != nil {
		return nil, err
	}
	resID := uuid.New()
	items, total := buildReservationItems(resID, req.Seats)
	currency := req.Currency
	if currency == "" {
		currency = "IDR"
	}
	ref, _ := generateBookingRef()
	res := &model.Reservation{ID: resID, UserID: req.UserID, EventID: req.EventID, BookingRef: ref, Status: model.StatusLocked, TotalAmount: total, Currency: currency, ExpiresAt: time.Now().Add(s.ttl)}
	if err := s.repo.CreateWithItems(ctx, res, items); err != nil {
		_ = s.locker.ReleaseSeats(ctx, eID, acquired)
		return nil, fmt.Errorf("failed to persist reservation: %w", err)
	}
	return toReservationResponse(res), nil
}
