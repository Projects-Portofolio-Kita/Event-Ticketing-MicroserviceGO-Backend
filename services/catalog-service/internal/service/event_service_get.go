// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: event_service_get.go
// Usability: Implements GetEventByID with cache-aside retrieval and detail response mapping.
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

func (s *eventService) GetEventByID(ctx context.Context, id uuid.UUID) (*dto.EventDetailResponse, error) {
	cacheKey := fmt.Sprintf("%s%s", EventKeyPrefix, id.String())
	if s.cacheService != nil {
		var cached dto.EventDetailResponse
		if s.cacheService.Get(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}
	detail := dto.ToEventDetailResponse(event)
	if s.cacheService != nil {
		_ = s.cacheService.Set(ctx, cacheKey, detail, DefaultCacheTTL)
	}
	return detail, nil
}
