// Package dto provides data transfer objects and response envelopes for the catalog service.
//
// File: event_response_dto.go
// Usability: Declares response structures for serialized public event summaries and full detail responses.
package dto

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

// EventResponse represents public summary information for an event
type EventResponse struct {
	ID          uuid.UUID         `json:"id"`
	Title       string            `json:"title"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	VenueID     uuid.UUID         `json:"venue_id"`
	Venue       *VenueResponse    `json:"venue,omitempty"`
	EventDate   time.Time         `json:"event_date"`
	DoorsOpen   *time.Time        `json:"doors_open,omitempty"`
	Status      model.EventStatus `json:"status"`
	BannerURL   string            `json:"banner_url,omitempty"`
	OrganizerID *uuid.UUID        `json:"organizer_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// EventDetailResponse represents full event information including venue and nested pricing tiers
type EventDetailResponse struct {
	ID           uuid.UUID             `json:"id"`
	Title        string                `json:"title"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	VenueID      uuid.UUID             `json:"venue_id"`
	Venue        VenueResponse         `json:"venue"`
	EventDate    time.Time             `json:"event_date"`
	DoorsOpen    *time.Time            `json:"doors_open,omitempty"`
	Status       model.EventStatus     `json:"status"`
	BannerURL    string                `json:"banner_url,omitempty"`
	OrganizerID  *uuid.UUID            `json:"organizer_id,omitempty"`
	PricingTiers []PricingTierResponse `json:"pricing_tiers"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}
