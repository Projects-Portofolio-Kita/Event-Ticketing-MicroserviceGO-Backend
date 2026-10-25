package repository

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReservationRepository struct {
	db *gorm.DB
}

func NewReservationRepository(db *gorm.DB) *ReservationRepository {
	return &ReservationRepository{db: db}
}

func (r *ReservationRepository) Create(reservation *model.Reservation) error {
	return r.db.Create(reservation).Error
}

func (r *ReservationRepository) FindByID(id uuid.UUID) (*model.Reservation, error) {
	var reservation model.Reservation

	err := r.db.Preload("Seats").First(&reservation, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &reservation, nil
}

func (r *ReservationRepository) Update(reservation *model.Reservation) error {
	return r.db.Save(reservation).Error
}

func (r *ReservationRepository) FindConflictingSeats(seatIDs []uuid.UUID) ([]uuid.UUID, error) {
	var conflicts []uuid.UUID

	err := r.db.
		Model(&model.ReservationSeat{}).
		Joins("JOIN reservations ON reservations.id = reservation_seats.reservation_id").
		Where("reservation_seats.seat_id IN ?", seatIDs).
		Where("reservations.status IN ?", []model.ReservationStatus{model.ReservationLocked, model.ReservationConfirmed}).
		Where("reservations.expires_at > ?", time.Now()).
		Pluck("reservation_seats.seat_id", &conflicts).Error

	if err != nil {
		return nil, err
	}

	return conflicts, nil
}

type SeatStatus struct {
	SeatID uuid.UUID `json:"seatId"`
	Status string    `json:"status"`
}

func (r *ReservationRepository) FindSeatStatusByEvent(eventID uuid.UUID) ([]SeatStatus, error) {
	var results []SeatStatus

	err := r.db.Raw(`
		SELECT rs.seat_id, r.status
		FROM reservation_seats rs
		JOIN reservations r ON r.id = rs.reservation_id
		WHERE r.event_id = ?
		  AND r.status IN ('LOCKED', 'CONFIRMED')
		  AND (r.status = 'CONFIRMED' OR r.expires_at > NOW())
	`, eventID).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
