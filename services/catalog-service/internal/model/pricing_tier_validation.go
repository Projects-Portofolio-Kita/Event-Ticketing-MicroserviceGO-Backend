// Package model defines domain entities, validation rules, and GORM database mappings.
//
// File: pricing_tier_validation.go
// Usability: Implements input integrity validation and GORM BeforeCreate lifecycle hooks for pricing tiers.
package model

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
