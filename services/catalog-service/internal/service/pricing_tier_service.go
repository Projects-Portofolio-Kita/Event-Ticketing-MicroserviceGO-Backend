package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
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
func NewPricingTierService(
	tierRepo repository.PricingTierRepository,
	eventRepo repository.EventRepository,
	cacheService CacheService,
) PricingTierService {
	return &pricingTierService{
		tierRepo:     tierRepo,
		eventRepo:    eventRepo,
		cacheService: cacheService,
	}
}

// AddPricingTier validates event status (rejects if ON_SALE/SOLD_OUT/terminal), creates tier, and invalidates cache
func (s *pricingTierService) AddPricingTier(ctx context.Context, eventID uuid.UUID, req dto.CreatePricingTierRequest) (*dto.PricingTierResponse, error) {
	// Validate tier fields upfront
	tierName := strings.TrimSpace(req.TierName)
	if tierName == "" {
		return nil, model.ErrInvalidTierName
	}
	if req.Price < 0 {
		return nil, model.ErrInvalidTierPrice
	}
	if req.TotalSeats <= 0 {
		return nil, model.ErrInvalidTierSeats
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "IDR"
	}

	// Fetch event to inspect status
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}

	// Check tier lock rule: cannot modify/add tiers if ON_SALE, SOLD_OUT, or terminal
	if !event.CanModifyPricingTiers() {
		return nil, model.ErrPricingTierLocked
	}

	tier := &model.PricingTier{
		ID:         uuid.New(),
		EventID:    eventID,
		TierName:   tierName,
		Price:      req.Price,
		Currency:   currency,
		TotalSeats: req.TotalSeats,
	}

	if err := s.tierRepo.Create(ctx, tier); err != nil {
		return nil, err
	}

	// Invalidate event detail and catalog cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateEvent(ctx, eventID)
	}

	return dto.ToPricingTierResponse(tier), nil
}

// GetPricingTiers retrieves all pricing tiers for an event, with cache-aside support
func (s *pricingTierService) GetPricingTiers(ctx context.Context, eventID uuid.UUID) ([]dto.PricingTierResponse, error) {
	// Verify event existence
	if _, err := s.eventRepo.GetByID(ctx, eventID); err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}

	tiersKey := fmt.Sprintf("%s%s:tiers", EventKeyPrefix, eventID.String())

	// 1. Check cache
	if s.cacheService != nil {
		var cached []dto.PricingTierResponse
		if s.cacheService.Get(ctx, tiersKey, &cached) {
			return cached, nil
		}
	}

	// 2. Query repository
	tiers, err := s.tierRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.PricingTierResponse, len(tiers))
	for i := range tiers {
		responses[i] = *dto.ToPricingTierResponse(&tiers[i])
	}

	// 3. Cache result
	if s.cacheService != nil {
		_ = s.cacheService.Set(ctx, tiersKey, responses, DefaultCacheTTL)
	}

	return responses, nil
}

// GetPricingTiersByEventID is an alias for GetPricingTiers
func (s *pricingTierService) GetPricingTiersByEventID(ctx context.Context, eventID uuid.UUID) ([]dto.PricingTierResponse, error) {
	return s.GetPricingTiers(ctx, eventID)
}
