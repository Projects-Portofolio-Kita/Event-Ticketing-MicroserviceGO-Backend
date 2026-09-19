package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReservationStatus string

const (
	StatusLocked    ReservationStatus = "LOCKED"
	StatusConfirmed ReservationStatus = "CONFIRMED"
	StatusExpired   ReservationStatus = "EXPIRED"
	StatusCancelled ReservationStatus = "CANCELLED"
)

type Reservation struct {
	ID          uuid.UUID         `gorm:"type:uuid;primaryKey" json:"reservationId"`
	UserID      uuid.UUID         `gorm:"type:uuid;not null;index" json:"userId"`
	EventID     uuid.UUID         `gorm:"type:uuid;not null;index" json:"eventId"`
	BookingRef  string            `gorm:"type:varchar(32);uniqueIndex;not null" json:"bookingRef"`
	Status      ReservationStatus `gorm:"type:varchar(20);not null;default:'LOCKED';index" json:"status"`
	TotalAmount float64           `gorm:"type:numeric(12,2);not null" json:"totalAmount"`
	Currency    string            `gorm:"type:varchar(5);default:'IDR'" json:"currency"`
	ExpiresAt   time.Time         `gorm:"not null;index" json:"expiresAt"`
	Items       []ReservationItem `gorm:"foreignKey:ReservationID;constraint:OnDelete:CASCADE" json:"seats"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt    `gorm:"index" json:"-"`
}

type ReservationItem struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ReservationID uuid.UUID `gorm:"type:uuid;not null;index" json:"reservationId"`
	SeatID        uuid.UUID `gorm:"type:uuid;not null;index" json:"seatId"`
	Section       string    `gorm:"type:varchar(50)" json:"section"`
	Row           string    `gorm:"type:varchar(20)" json:"row"`
	Number        string    `gorm:"type:varchar(20)" json:"number"`
	Price         float64   `gorm:"type:numeric(12,2);not null" json:"price"`
	CreatedAt     time.Time `json:"createdAt"`
}
