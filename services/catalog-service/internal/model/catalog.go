package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Code      string    `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Venue     string    `gorm:"type:varchar(255);not null" json:"venue"`
	EventDate time.Time `json:"eventDate"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

type SeatCategory struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	EventID   uuid.UUID `gorm:"type:uuid;not null;index" json:"eventId"`
	Code      string    `gorm:"type:varchar(50);not null" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Subtitle  string    `gorm:"type:varchar(255);not null" json:"subtitle"`
	DotColor  string    `gorm:"type:varchar(20);not null" json:"dotColor"`
	Price     float64   `gorm:"type:decimal(12,2);not null" json:"price"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
}

func (c *SeatCategory) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type Seat struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"seatId"`
	EventID    uuid.UUID `gorm:"type:uuid;not null;index" json:"eventId"`
	CategoryID uuid.UUID `gorm:"type:uuid;not null;index" json:"categoryId"`
	Label      string    `gorm:"type:varchar(50);not null" json:"label"`
	RowName    string    `gorm:"type:varchar(10);not null" json:"rowName"`
	Number     int       `gorm:"not null" json:"number"`
	Side       string    `gorm:"type:varchar(10);not null" json:"side"`
	SortOrder  int       `gorm:"not null;default:0" json:"sortOrder"`
}

func (s *Seat) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
