package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SaleMode decides how tickets are sold: a numbered seat map (SEATED) or
// quota-based ticket tiers such as VIP/VVIP/Regular (TIER).
type SaleMode string

const (
	SaleModeSeated SaleMode = "SEATED"
	SaleModeTier   SaleMode = "TIER"
)

// Event categories used for discovery filters.
const (
	EventCategoryConcert  = "concert"
	EventCategoryFestival = "festival"
	EventCategorySports   = "sports"
	EventCategorySeminar  = "seminar"
	EventCategoryTheater  = "theater"
	EventCategoryWorkshop = "workshop"
	EventCategoryOther    = "other"
)

// ValidEventCategories is the whitelist the create/list endpoints accept.
var ValidEventCategories = map[string]bool{
	EventCategoryConcert:  true,
	EventCategoryFestival: true,
	EventCategorySports:   true,
	EventCategorySeminar:  true,
	EventCategoryTheater:  true,
	EventCategoryWorkshop: true,
	EventCategoryOther:    true,
}

type Event struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Code      string    `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Venue     string    `gorm:"type:varchar(255);not null" json:"venue"`
	EventDate time.Time `json:"eventDate"`
	// Discovery metadata: the location picker and category menu filter on these.
	City                string   `gorm:"type:varchar(100);not null;default:'Jakarta'" json:"city"`
	Category            string   `gorm:"type:varchar(50);not null;default:'concert'" json:"category"`
	Description         string   `gorm:"type:text;not null;default:''" json:"description,omitempty"`
	PosterURL           string   `gorm:"type:varchar(512);not null;default:''" json:"posterUrl,omitempty"`
	SaleMode            SaleMode `gorm:"type:varchar(10);not null;default:'SEATED'" json:"saleMode"`
	SeatRows            int      `gorm:"not null;default:4" json:"seatRows"`
	SeatsPerRow         int      `gorm:"not null;default:16" json:"seatsPerRow"`
	LeftBlockSize       int      `gorm:"not null;default:8" json:"leftBlockSize"`
	PayoutBankName      string   `gorm:"type:varchar(100);not null;default:''" json:"payoutBankName,omitempty"`
	PayoutAccountNumber string   `gorm:"type:varchar(50);not null;default:''" json:"payoutAccountNumber,omitempty"`
	PayoutAccountName   string   `gorm:"type:varchar(100);not null;default:''" json:"payoutAccountName,omitempty"`
	// CreatedBy is the user (access-token subject) who posted the event.
	// Seeded events have no poster, hence the pointer.
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"createdBy,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
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
	// Quota > 0 only for TIER events (how many tickets of this type exist).
	// SEATED categories keep 0 — availability comes from the seat map.
	Quota       int    `gorm:"not null;default:0" json:"quota"`
	Description string `gorm:"type:text;not null;default:''" json:"description,omitempty"`
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
