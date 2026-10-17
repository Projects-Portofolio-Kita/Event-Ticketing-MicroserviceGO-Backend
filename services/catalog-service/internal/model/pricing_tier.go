package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidTierPrice = errors.New("pricing tier price must be greater than or equal to 0")
	ErrInvalidTierSeats = errors.New("pricing tier total seats must be greater than 0")
	ErrInvalidTierName  = errors.New("pricing tier name cannot be empty")
	ErrNilPricingTier   = errors.New("pricing tier receiver is nil")
)

// PricingTier represents a seating category and pricing level for an event
type PricingTier struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	EventID    uuid.UUID `gorm:"type:uuid;not null;index" json:"event_id"`
	TierName   string    `gorm:"type:varchar(100);not null" json:"tier_name"`
	Price      float64   `gorm:"type:decimal(12,2);not null" json:"price"`
	Currency   string    `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	TotalSeats int       `gorm:"not null" json:"total_seats"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name for PricingTier
func (PricingTier) TableName() string {
	return "pricing_tiers"
}

// Validate validates that price >= 0, total_seats > 0, and tier name is not empty
func (pt *PricingTier) Validate() error {
	if pt == nil {
		return ErrNilPricingTier
	}
	if strings.TrimSpace(pt.TierName) == "" {
		return ErrInvalidTierName
	}
	if pt.Price < 0 {
		return ErrInvalidTierPrice
	}
	if pt.TotalSeats <= 0 {
		return ErrInvalidTierSeats
	}
	if pt.Currency == "" {
		pt.Currency = "IDR"
	}
	return nil
}

// BeforeCreate validates the tier and ensures UUID and default currency are assigned
func (pt *PricingTier) BeforeCreate(tx *gorm.DB) error {
	if pt == nil {
		return ErrNilPricingTier
	}
	if pt.ID == uuid.Nil {
		pt.ID = uuid.New()
	}
	if pt.Currency == "" {
		pt.Currency = "IDR"
	}
	return pt.Validate()
}
