package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReservationStatus string

const (
	ReservationLocked    ReservationStatus = "LOCKED"
	ReservationConfirmed ReservationStatus = "CONFIRMED"
	ReservationExpired   ReservationStatus = "EXPIRED"
	ReservationCancelled ReservationStatus = "CANCELLED"
)

type Reservation struct {
	ID          uuid.UUID         `gorm:"type:uuid;primaryKey" json:"reservationId"`
	UserID      uuid.UUID         `gorm:"type:uuid;not null;index" json:"userId"`
	EventID     uuid.UUID         `gorm:"type:uuid;not null;index" json:"eventId"`
	BookingRef  string            `gorm:"type:varchar(50);uniqueIndex" json:"bookingRef"`
	Status      ReservationStatus `gorm:"type:varchar(20);not null;default:'LOCKED'" json:"status"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	TotalAmount float64           `gorm:"type:decimal(12,2);not null" json:"totalAmount"`
	Currency    string            `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	Seats       []ReservationSeat `gorm:"foreignKey:ReservationID;constraint:OnDelete:CASCADE" json:"seats"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func (r *Reservation) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type ReservationSeat struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ReservationID uuid.UUID `gorm:"type:uuid;not null;index" json:"reservationId"`
	SeatID        uuid.UUID `gorm:"type:uuid;not null;index" json:"seatId"`
	Section       string    `gorm:"type:varchar(100)" json:"section"`
	Row           string    `gorm:"type:varchar(10)" json:"row"`
	Number        string    `gorm:"type:varchar(10)" json:"number"`
	Price         float64   `gorm:"type:decimal(12,2)" json:"price"`
}

func (s *ReservationSeat) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
