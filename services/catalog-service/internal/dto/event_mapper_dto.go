// Package dto provides data transfer objects and response envelopes for the catalog service.
//
// File: event_mapper_dto.go
// Usability: Provides domain model to DTO transformation mappers (ToEventResponse, ToEventDetailResponse).
package dto

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

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
		ID: e.ID, Title: e.Name, Name: e.Name, Description: e.Description,
		VenueID: e.VenueID, Venue: venueResp, EventDate: e.EventDate,
		Status: e.Status, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
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
		ID: e.ID, Title: e.Name, Name: e.Name, Description: e.Description,
		VenueID: e.VenueID, Venue: venueResp, EventDate: e.EventDate,
		Status: e.Status, PricingTiers: tierResponses, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}
