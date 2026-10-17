package dto

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

// CreateEventRequest represents the request payload to create a new event
type CreateEventRequest struct {
	Title       string     `json:"title" binding:"required,min=3,max=255"`
	Name        string     `json:"name,omitempty"` // Alias for Title
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

// VenueResponse represents public venue information returned in event responses
type VenueResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	TotalCapacity int       `json:"total_capacity"`
	LayoutConfig  string    `json:"layout_config,omitempty"`
}

// EventResponse represents public summary information for an event
type EventResponse struct {
	ID          uuid.UUID         `json:"id"`
	Title       string            `json:"title"`
	Name        string            `json:"name"` // Alias for compatibility
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

// ToVenueResponse maps model.Venue to VenueResponse
func ToVenueResponse(v *model.Venue) *VenueResponse {
	if v == nil {
		return nil
	}
	return &VenueResponse{
		ID:            v.ID,
		Name:          v.Name,
		Address:       v.Address,
		City:          v.City,
		TotalCapacity: v.TotalCapacity,
		LayoutConfig:  v.LayoutConfig,
	}
}

// ToEventResponse maps model.Event to EventResponse
func ToEventResponse(e *model.Event) *EventResponse {
	if e == nil {
		return nil
	}
	var venueResp *VenueResponse
	if e.Venue.ID != uuid.Nil {
		venueResp = ToVenueResponse(&e.Venue)
	}
	return &EventResponse{
		ID:          e.ID,
		Title:       e.Name,
		Name:        e.Name,
		Description: e.Description,
		VenueID:     e.VenueID,
		Venue:       venueResp,
		EventDate:   e.EventDate,
		Status:      e.Status,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// ToEventDetailResponse maps model.Event to EventDetailResponse with preloaded Venue and Tiers
func ToEventDetailResponse(e *model.Event) *EventDetailResponse {
	if e == nil {
		return nil
	}
	var venueResp VenueResponse
	if e.Venue.ID != uuid.Nil {
		venueResp = *ToVenueResponse(&e.Venue)
	}
	tierResponses := make([]PricingTierResponse, len(e.PricingTiers))
	for i := range e.PricingTiers {
		tierResponses[i] = *ToPricingTierResponse(&e.PricingTiers[i])
	}
	return &EventDetailResponse{
		ID:           e.ID,
		Title:        e.Name,
		Name:         e.Name,
		Description:  e.Description,
		VenueID:      e.VenueID,
		Venue:        venueResp,
		EventDate:    e.EventDate,
		Status:       e.Status,
		PricingTiers: tierResponses,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
