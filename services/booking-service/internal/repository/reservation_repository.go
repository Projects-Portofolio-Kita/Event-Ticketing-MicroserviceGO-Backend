package repository

import (
	"fmt"
	"sort"
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

// CreateTier persists a TIER reservation atomically. Inside one transaction it
// takes a Postgres advisory lock per tier (sorted, so overlapping baskets can
// never deadlock), re-counts what is already locked/confirmed for those tiers
// and only then runs the caller's quota check before inserting. That is what
// makes "the last ticket" safe under concurrent buyers: everyone serializes on
// the same lock key and re-reads the fresh count before deciding.
func (r *ReservationRepository) CreateTier(reservation *model.Reservation, verify func(reserved map[uuid.UUID]int) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		categoryIDs := make([]uuid.UUID, 0, len(reservation.Items))
		for _, item := range reservation.Items {
			categoryIDs = append(categoryIDs, item.CategoryID)
		}
		sort.Slice(categoryIDs, func(i, j int) bool {
			return categoryIDs[i].String() < categoryIDs[j].String()
		})

		for _, categoryID := range categoryIDs {
			lockKey := fmt.Sprintf("tier:%s:%s", reservation.EventID, categoryID)
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
				return err
			}
		}

		reserved, err := sumReservedByCategory(tx, reservation.EventID, categoryIDs)
		if err != nil {
			return err
		}

		// The quota check runs against the counts read under the lock.
		if err := verify(reserved); err != nil {
			return err
		}

		return tx.Create(reservation).Error
	})
}

func (r *ReservationRepository) FindByID(id uuid.UUID) (*model.Reservation, error) {
	var reservation model.Reservation

	err := r.db.Preload("Seats").Preload("Items").First(&reservation, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &reservation, nil
}

// FindByUser lists the caller's reservations, newest first, for "Tiket Saya".
func (r *ReservationRepository) FindByUser(userID uuid.UUID, limit int) ([]model.Reservation, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var reservations []model.Reservation
	err := r.db.
		Preload("Seats").
		Preload("Items").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&reservations).Error
	if err != nil {
		return nil, err
	}

	return reservations, nil
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

// TicketStatus is how many tickets of one tier are already locked or sold.
// catalog-service reads it to render "sisa tiket" on tier cards.
type TicketStatus struct {
	CategoryID uuid.UUID `json:"categoryId"`
	Reserved   int       `json:"reserved"`
}

// FindTicketStatusByEvent reports the reserved quantity per tier for one
// event: the availability feed consumed by catalog-service.
func (r *ReservationRepository) FindTicketStatusByEvent(eventID uuid.UUID) ([]TicketStatus, error) {
	reserved, err := sumReservedByCategory(r.db, eventID, nil)
	if err != nil {
		return nil, err
	}

	statuses := make([]TicketStatus, 0, len(reserved))
	for categoryID, quantity := range reserved {
		statuses = append(statuses, TicketStatus{CategoryID: categoryID, Reserved: quantity})
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].CategoryID.String() < statuses[j].CategoryID.String()
	})

	return statuses, nil
}

// sumReservedByCategory adds up the quantities locked or sold per tier of an
// event. Expired LOCKED reservations stop counting (their seats free up on
// read), CONFIRMED ones always count. Passing no category ids counts all of
// the event's tiers.
func sumReservedByCategory(db *gorm.DB, eventID uuid.UUID, categoryIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	query := db.Model(&model.ReservationItem{}).
		Select("reservation_items.category_id, COALESCE(SUM(reservation_items.quantity), 0) AS reserved").
		Joins("JOIN reservations ON reservations.id = reservation_items.reservation_id").
		Where("reservations.event_id = ?", eventID).
		Where("reservations.status IN ?", []model.ReservationStatus{model.ReservationLocked, model.ReservationConfirmed}).
		Where("reservations.status = ? OR reservations.expires_at > ?", model.ReservationConfirmed, time.Now()).
		Group("reservation_items.category_id")
	if len(categoryIDs) > 0 {
		query = query.Where("reservation_items.category_id IN ?", categoryIDs)
	}

	var rows []TicketStatus
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	reserved := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		reserved[row.CategoryID] = row.Reserved
	}

	return reserved, nil
}
