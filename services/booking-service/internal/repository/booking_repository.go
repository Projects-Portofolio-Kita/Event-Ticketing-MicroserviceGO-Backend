package repository

import (
	"context"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookingRepository struct {
	db *gorm.DB
}

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

// FindByID retrieves a reservation by its UUID along with all reserved seats.
func (r *BookingRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Reservation, error) {
	var reservation model.Reservation
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ?", id).
		First(&reservation).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

// FindByBookingRef retrieves a reservation by its human-readable reference code.
func (r *BookingRepository) FindByBookingRef(ctx context.Context, ref string) (*model.Reservation, error) {
	var reservation model.Reservation
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("booking_ref = ?", ref).
		First(&reservation).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

// UpdateStatus updates the reservation status.
func (r *BookingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ReservationStatus) error {
	return r.db.WithContext(ctx).
		Model(&model.Reservation{}).
		Where("id = ?", id).
		Update("status", status).Error
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
