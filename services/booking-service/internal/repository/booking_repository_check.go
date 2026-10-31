// Package repository handles persistence and database operations for bookings.
//
// File: booking_repository_check.go
// Usability: Provides CheckSeatsAlreadyBooked to verify if seats are confirmed or currently locked.
package repository

import (
	"context"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
)

// CheckSeatsAlreadyBooked verifies if any of the requested seats are currently confirmed or actively locked in the database.
func (r *BookingRepository) CheckSeatsAlreadyBooked(ctx context.Context, eventID uuid.UUID, seatIDs []uuid.UUID) (bool, error) {
	now := time.Now()
	var count int64
	err := r.db.WithContext(ctx).
		Table("reservation_items").
		Joins("JOIN reservations ON reservations.id = reservation_items.reservation_id").
		Where("reservations.event_id = ? AND reservation_items.seat_id IN (?)", eventID, seatIDs).
		Where("reservations.status = ? OR (reservations.status = ? AND reservations.expires_at > ?)",
			model.StatusConfirmed, model.StatusLocked, now).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
