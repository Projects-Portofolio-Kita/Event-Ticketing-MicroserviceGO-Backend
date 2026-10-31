// Package model defines domain entities, validation rules, and GORM database mappings.
//
// File: event.go
// Usability: Declares Event domain model entity, database relationships to Venues and Tiers, and BeforeCreate hooks.
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Event represents an event in the catalog with lifecycle status and pricing tiers
type Event struct {
	ID           uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string        `gorm:"type:varchar(255);not null" json:"name"`
	Description  string        `gorm:"type:text" json:"description"`
	VenueID      uuid.UUID     `gorm:"type:uuid;not null;index" json:"venue_id"`
	Venue        Venue         `gorm:"foreignKey:VenueID;references:ID" json:"venue,omitempty"`
	Status       EventStatus   `gorm:"type:varchar(50);not null;default:'DRAFT';index" json:"status"`
	EventDate    time.Time     `gorm:"not null;index" json:"event_date"`
	StartDate    time.Time     `json:"start_date"`
	EndDate      time.Time     `json:"end_date"`
	PricingTiers []PricingTier `gorm:"foreignKey:EventID;references:ID;constraint:OnDelete:CASCADE" json:"pricing_tiers,omitempty"`
	CreatedAt    time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name for Event
func (Event) TableName() string {
	return "events"
}

// BeforeCreate generates a UUID if not provided and sets initial status to DRAFT
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e == nil {
		return ErrNilEvent
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Status == "" {
		e.Status = StatusDraft
	}
	return nil
}
