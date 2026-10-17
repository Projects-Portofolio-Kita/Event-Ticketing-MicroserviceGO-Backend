package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Venue represents a physical location where events take place
type Venue struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	Address       string    `gorm:"type:text;not null" json:"address"`
	City          string    `gorm:"type:varchar(100);not null;index" json:"city"`
	TotalCapacity int       `gorm:"not null" json:"total_capacity"`
	LayoutConfig  string    `gorm:"type:text" json:"layout_config"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name for Venue
func (Venue) TableName() string {
	return "venues"
}

// BeforeCreate generates a UUID if one is not already provided
func (v *Venue) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
