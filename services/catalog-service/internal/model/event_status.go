// Package model defines domain entities, validation rules, and GORM database mappings.
//
// File: event_status.go
// Usability: Declares EventStatus lifecycle enum, domain errors, and the allowed FSM state transition matrix.
package model

import "errors"

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

var allowedTransitions = map[EventStatus][]EventStatus{
	StatusDraft:     {StatusPublished, StatusCancelled},
	StatusPublished: {StatusOnSale, StatusCancelled},
	StatusOnSale:    {StatusSoldOut, StatusCancelled, StatusPast},
	StatusSoldOut:   {StatusOnSale, StatusCancelled, StatusPast},
	StatusCancelled: {},
	StatusPast:      {},
}
