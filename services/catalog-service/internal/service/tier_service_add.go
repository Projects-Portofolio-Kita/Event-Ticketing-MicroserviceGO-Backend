// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: tier_service_add.go
// Usability: Implements AddPricingTier with event status validation, capacity checks, and cache invalidation.
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
)

func (s *pricingTierService) AddPricingTier(ctx context.Context, eventID uuid.UUID, req dto.CreatePricingTierRequest) (*dto.PricingTierResponse, error) {
	name := strings.TrimSpace(req.TierName)
	if name == "" { return nil, model.ErrInvalidTierName }
	if req.Price < 0 { return nil, model.ErrInvalidTierPrice }
	if req.TotalSeats <= 0 { return nil, model.ErrInvalidTierSeats }
	curr := strings.TrimSpace(req.Currency)
	if curr == "" { curr = "IDR" }
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) { return nil, model.ErrEventNotFound }
		return nil, err
	}
	if !event.CanModifyPricingTiers() { return nil, model.ErrPricingTierLocked }
	tier := &model.PricingTier{ID: uuid.New(), EventID: eventID, TierName: name, Price: req.Price, Currency: curr, TotalSeats: req.TotalSeats}
	if err := s.tierRepo.Create(ctx, tier); err != nil { return nil, err }
	if s.cacheService != nil { _ = s.cacheService.InvalidateEvent(ctx, eventID) }
	return dto.ToPricingTierResponse(tier), nil
}