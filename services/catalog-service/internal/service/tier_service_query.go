// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: tier_service_query.go
// Usability: Implements GetPricingTiers with Redis cache-aside querying and repository fallback.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
)

func (s *pricingTierService) GetPricingTiers(ctx context.Context, eventID uuid.UUID) ([]dto.PricingTierResponse, error) {
	if _, err := s.eventRepo.GetByID(ctx, eventID); err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}
	tiersKey := fmt.Sprintf("%s%s:tiers", EventKeyPrefix, eventID.String())
	if s.cacheService != nil {
		var cached []dto.PricingTierResponse
		if s.cacheService.Get(ctx, tiersKey, &cached) {
			return cached, nil
		}
	}
	tiers, err := s.tierRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	responses := make([]dto.PricingTierResponse, len(tiers))
	for i := range tiers {
		responses[i] = *dto.ToPricingTierResponse(&tiers[i])
	}
	if s.cacheService != nil {
		_ = s.cacheService.Set(ctx, tiersKey, responses, DefaultCacheTTL)
	}
	return responses, nil
}
