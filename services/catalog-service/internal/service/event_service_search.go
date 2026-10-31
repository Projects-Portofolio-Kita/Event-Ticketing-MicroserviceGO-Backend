// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: event_service_search.go
// Usability: Implements SearchEvents with multi-parameter filtering, pagination metadata, and 60s Redis caching.
package service

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
)

func (s *eventService) SearchEvents(ctx context.Context, filter dto.EventFilter, page, limit int) (*dto.PaginatedEventsResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	cacheKey := GenerateCatalogCacheKey(filter, page, limit)
	if s.cacheService != nil {
		var cached dto.PaginatedEventsResponse
		if s.cacheService.Get(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}
	events, total, err := s.eventRepo.List(ctx, filter, page, limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.EventResponse, len(events))
	for i := range events {
		items[i] = *dto.ToEventResponse(&events[i])
	}
	meta := dto.NewPaginationMeta(page, limit, total)
	res := &dto.PaginatedEventsResponse{Items: items, Data: items, Meta: meta}
	if s.cacheService != nil {
		_ = s.cacheService.Set(ctx, cacheKey, res, DefaultCacheTTL)
	}
	return res, nil
}
