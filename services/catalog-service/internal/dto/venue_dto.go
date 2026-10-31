// Package dto provides data transfer objects and response envelopes for the catalog service.
//
// File: venue_dto.go
// Usability: Declares VenueResponse DTO and ToVenueResponse mapper function.
package dto

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

// VenueResponse represents public venue information returned in event responses
type VenueResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	TotalCapacity int       `json:"total_capacity"`
	LayoutConfig  string    `json:"layout_config,omitempty"`
}

// ToVenueResponse maps model.Venue to VenueResponse
func ToVenueResponse(v *model.Venue) *VenueResponse {
	if v == nil {
		return nil
	}
	return &VenueResponse{
		ID: v.ID, Name: v.Name, Address: v.Address, City: v.City,
		TotalCapacity: v.TotalCapacity, LayoutConfig: v.LayoutConfig,
	}
}
