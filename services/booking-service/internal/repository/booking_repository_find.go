// Package repository handles persistence and database operations for bookings.
//
// File: booking_repository_find.go
// Usability: Provides query methods to find reservations by ID, booking reference, and expired status.
package repository

import (
	"context"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
)

// FindByID retrieves a reservation by its UUID along with all reserved seats.
func (r *BookingRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Reservation, error) {
	var reservation model.Reservation
	err := r.db.WithContext(ctx).Preload("Items").Where("id = ?", id).First(&reservation).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

// FindByBookingRef retrieves a reservation by its human-readable reference code.
func (r *BookingRepository) FindByBookingRef(ctx context.Context, ref string) (*model.Reservation, error) {
	var reservation model.Reservation
	err := r.db.WithContext(ctx).Preload("Items").Where("booking_ref = ?", ref).First(&reservation).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

// FindExpiredLockedReservations finds all LOCKED reservations whose expiration time has passed.
func (r *BookingRepository) FindExpiredLockedReservations(ctx context.Context, limit int) ([]model.Reservation, error) {
	var reservations []model.Reservation
	now := time.Now()
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("status = ? AND expires_at < ?", model.StatusLocked, now).
		Limit(limit).
		Find(&reservations).Error
	return reservations, err
}
