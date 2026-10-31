// Package repository handles persistence and database operations for bookings.
//
// File: booking_repository.go
// Usability: Defines BookingRepository struct, constructor, transactional reservation creation, and status updates.
package repository

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BookingRepository provides GORM database access for reservations.
type BookingRepository struct {
	db *gorm.DB
}

// NewBookingRepository creates a new BookingRepository.
func NewBookingRepository(db *gorm.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// CreateWithItems saves a new reservation and its items in an atomic database transaction.
func (r *BookingRepository) CreateWithItems(ctx context.Context, reservation *model.Reservation, items []model.ReservationItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(reservation).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ReservationID = reservation.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		reservation.Items = items
		return nil
	})
}

// UpdateStatus updates the reservation status.
func (r *BookingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ReservationStatus) error {
	return r.db.WithContext(ctx).
		Model(&model.Reservation{}).
		Where("id = ?", id).
		Update("status", status).Error
}
