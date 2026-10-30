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

// ReservationMode mirrors catalog's SaleMode: a numbered seat map (SEATED) or
// quota-based ticket tiers such as VIP/VVIP/Regular (TIER).
type ReservationMode string

const (
	ReservationModeSeated ReservationMode = "SEATED"
	ReservationModeTier   ReservationMode = "TIER"
)

type Reservation struct {
	ID         uuid.UUID         `gorm:"type:uuid;primaryKey" json:"reservationId"`
	UserID     uuid.UUID         `gorm:"type:uuid;not null;index" json:"userId"`
	EventID    uuid.UUID         `gorm:"type:uuid;not null;index" json:"eventId"`
	BookingRef string            `gorm:"type:varchar(50);uniqueIndex" json:"bookingRef"`
	Status     ReservationStatus `gorm:"type:varchar(20);not null;default:'LOCKED'" json:"status"`
	// Mode: SEATED reservations carry Seats, TIER reservations carry Items.
	Mode        ReservationMode   `gorm:"type:varchar(10);not null;default:'SEATED'" json:"mode"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	TotalAmount float64           `gorm:"type:decimal(12,2);not null" json:"totalAmount"`
	Currency    string            `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	Seats       []ReservationSeat `gorm:"foreignKey:ReservationID;constraint:OnDelete:CASCADE" json:"seats"`
	Items       []ReservationItem `gorm:"foreignKey:ReservationID;constraint:OnDelete:CASCADE" json:"items"`
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

// ReservationItem is one quota-based ticket line of a TIER reservation: how
// many tickets of one tier (category) are locked, at the price catalog-service
// reported (clients never send prices).
type ReservationItem struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ReservationID uuid.UUID `gorm:"type:uuid;not null;index" json:"reservationId"`
	CategoryID    uuid.UUID `gorm:"type:uuid;not null;index" json:"categoryId"`
	CategoryName  string    `gorm:"type:varchar(100);not null;default:''" json:"categoryName"`
	Quantity      int       `gorm:"not null" json:"quantity"`
	UnitPrice     float64   `gorm:"type:decimal(12,2);not null" json:"unitPrice"`
	Subtotal      float64   `gorm:"type:decimal(12,2);not null" json:"subtotal"`
}

func (i *ReservationItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
