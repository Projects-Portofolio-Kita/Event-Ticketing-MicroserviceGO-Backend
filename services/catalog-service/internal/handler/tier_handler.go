// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: tier_handler.go
// Usability: Declares PricingTierHandler struct, constructor, and JSON response envelopes.
package handler

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
)

// PricingTierHandler handles HTTP transport for event pricing tiers
type PricingTierHandler struct {
	tierService service.PricingTierService
}

// NewPricingTierHandler creates a new PricingTierHandler instance
func NewPricingTierHandler(tierService service.PricingTierService) *PricingTierHandler {
	return &PricingTierHandler{tierService: tierService}
}

type pricingTierResponseEnvelope struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message,omitempty"`
	Data    *dto.PricingTierResponse `json:"data,omitempty"`
	*dto.PricingTierResponse
}

type pricingTiersResponseEnvelope struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message,omitempty"`
	Data    []dto.PricingTierResponse `json:"data"`
	Tiers   []dto.PricingTierResponse `json:"tiers,omitempty"`
}
