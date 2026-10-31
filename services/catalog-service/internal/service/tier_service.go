// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: tier_service.go
// Usability: Declares PricingTierService interface, implementation struct, constructor, and alias lookups.
package service

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
)

// PricingTierService defines business operations for managing pricing tiers
type PricingTierService interface {
	AddPricingTier(ctx context.Context, eventID uuid.UUID, req dto.CreatePricingTierRequest) (*dto.PricingTierResponse, error)
	GetPricingTiers(ctx context.Context, eventID uuid.UUID) ([]dto.PricingTierResponse, error)
	GetPricingTiersByEventID(ctx context.Context, eventID uuid.UUID) ([]dto.PricingTierResponse, error)
}

type pricingTierService struct {
	tierRepo     repository.PricingTierRepository
	eventRepo    repository.EventRepository
	cacheService CacheService
}

// NewPricingTierService initializes a new PricingTierService
func NewPricingTierService(tierRepo repository.PricingTierRepository, eventRepo repository.EventRepository, cacheService CacheService) PricingTierService {
	return &pricingTierService{
		tierRepo:     tierRepo,
		eventRepo:    eventRepo,
		cacheService: cacheService,
	}
}

// GetPricingTiersByEventID is an alias for GetPricingTiers
func (s *pricingTierService) GetPricingTiersByEventID(ctx context.Context, eventID uuid.UUID) ([]dto.PricingTierResponse, error) {
	return s.GetPricingTiers(ctx, eventID)
}
