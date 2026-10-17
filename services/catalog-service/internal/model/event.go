package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventStatus defines the allowed lifecycle states of an Event
type EventStatus string

const (
	StatusDraft     EventStatus = "DRAFT"
	StatusPublished EventStatus = "PUBLISHED"
	StatusOnSale    EventStatus = "ON_SALE"
	StatusSoldOut   EventStatus = "SOLD_OUT"
	StatusCancelled EventStatus = "CANCELLED"
	StatusPast      EventStatus = "PAST"
)

var (
	ErrInvalidStateTransition = errors.New("invalid event state transition")
	ErrTerminalState          = errors.New("event is in a terminal state and cannot be modified")
	ErrEventNotFound          = errors.New("event not found")
	ErrPricingTierLocked      = errors.New("pricing tiers cannot be modified for event in current status")
	ErrNilEvent               = errors.New("event receiver is nil")
)

// allowedTransitions defines the strict FSM transitions for an Event
var allowedTransitions = map[EventStatus][]EventStatus{
	StatusDraft:     {StatusPublished, StatusCancelled},
	StatusPublished: {StatusOnSale, StatusCancelled},
	StatusOnSale:    {StatusSoldOut, StatusCancelled, StatusPast},
	StatusSoldOut:   {StatusOnSale, StatusCancelled, StatusPast},
	StatusCancelled: {},
	StatusPast:      {},
}

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

// CanTransitionTo checks whether transitioning to next is valid according to the FSM matrix
func (e *Event) CanTransitionTo(next EventStatus) bool {
	if e == nil {
		return false
	}
	targets, ok := allowedTransitions[e.Status]
	if !ok {
		return false
	}
	for _, target := range targets {
		if target == next {
			return true
		}
	}
	return false
}

// TransitionTo applies the state transition if allowed, returning an appropriate domain error if not
func (e *Event) TransitionTo(next EventStatus) error {
	if e == nil {
		return ErrNilEvent
	}
	if e.IsTerminal() {
		return ErrTerminalState
	}
	if !e.CanTransitionTo(next) {
		return ErrInvalidStateTransition
	}
	e.Status = next
	return nil
}

// IsTerminal returns true if the event is in a terminal state (CANCELLED or PAST)
func (e *Event) IsTerminal() bool {
	if e == nil {
		return false
	}
	return e.Status == StatusCancelled || e.Status == StatusPast
}

// CanModifyPricingTiers returns true only if the event is in DRAFT or PUBLISHED status
func (e *Event) CanModifyPricingTiers() bool {
	if e == nil {
		return false
	}
	return e.Status == StatusDraft || e.Status == StatusPublished
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
