// Package dto provides data transfer objects and response envelopes for the catalog service.
//
// File: event_request_dto.go
// Usability: Declares request structures and input validation bindings for creating and updating events.
package dto

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

// CreateEventRequest represents the request payload to create a new event
type CreateEventRequest struct {
	Title       string     `json:"title" binding:"required,min=3,max=255"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description"`
	VenueID     uuid.UUID  `json:"venue_id" binding:"required"`
	EventDate   time.Time  `json:"event_date" binding:"required"`
	DoorsOpen   *time.Time `json:"doors_open,omitempty"`
	BannerURL   string     `json:"banner_url,omitempty"`
	OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
}

// GetTitle returns the effective title/name from the request
func (r *CreateEventRequest) GetTitle() string {
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

// UpdateEventStatusRequest represents request payload to transition event status
type UpdateEventStatusRequest struct {
	Status model.EventStatus `json:"status" binding:"required"`
}

// UpdateEventRequest represents request payload to update general event details
type UpdateEventRequest struct {
	Title       *string    `json:"title,omitempty"`
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	VenueID     *uuid.UUID `json:"venue_id,omitempty"`
	EventDate   *time.Time `json:"event_date,omitempty"`
	DoorsOpen   *time.Time `json:"doors_open,omitempty"`
	BannerURL   *string    `json:"banner_url,omitempty"`
}
