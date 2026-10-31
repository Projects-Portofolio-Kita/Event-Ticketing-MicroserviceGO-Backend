// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: event_service.go
// Usability: Declares EventService interface, error definitions, eventService struct, and constructor.
package service

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrVenueNotFound          = repository.ErrVenueNotFound
	ErrEventNotFound          = model.ErrEventNotFound
	ErrPricingTierLocked      = model.ErrPricingTierLocked
	ErrInvalidStateTransition = model.ErrInvalidStateTransition
	ErrTerminalState          = model.ErrTerminalState
)

// EventService defines business operations on events
type EventService interface {
	CreateEvent(ctx context.Context, req dto.CreateEventRequest) (*dto.EventResponse, error)
	GetEventByID(ctx context.Context, id uuid.UUID) (*dto.EventDetailResponse, error)
	UpdateEventStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) (*dto.EventResponse, error)
	SearchEvents(ctx context.Context, filter dto.EventFilter, page, limit int) (*dto.PaginatedEventsResponse, error)
	GetEvents(ctx context.Context, filter dto.EventFilter, page, limit int) (*dto.PaginatedEventsResponse, error)
}

type eventService struct {
	eventRepo    repository.EventRepository
	venueRepo    repository.VenueRepository
	cacheService CacheService
}

// NewEventService initializes a new EventService
func NewEventService(eventRepo repository.EventRepository, venueRepo repository.VenueRepository, cacheService CacheService) EventService {
	return &eventService{eventRepo: eventRepo, venueRepo: venueRepo, cacheService: cacheService}
}

func (s *eventService) GetEvents(ctx context.Context, filter dto.EventFilter, page, limit int) (*dto.PaginatedEventsResponse, error) {
	return s.SearchEvents(ctx, filter, page, limit)
}
