package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
)

// Common domain and service errors
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
func NewEventService(
	eventRepo repository.EventRepository,
	venueRepo repository.VenueRepository,
	cacheService CacheService,
) EventService {
	return &eventService{
		eventRepo:    eventRepo,
		venueRepo:    venueRepo,
		cacheService: cacheService,
	}
}

// CreateEvent validates venue existence, persists new event in DRAFT status, and invalidates catalog cache
func (s *eventService) CreateEvent(ctx context.Context, req dto.CreateEventRequest) (*dto.EventResponse, error) {
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, errors.New("event title is required")
	}

	if req.VenueID == uuid.Nil {
		return nil, errors.New("venue_id is required")
	}

	venue, err := s.venueRepo.GetByID(ctx, req.VenueID)
	if err != nil {
		if errors.Is(err, repository.ErrVenueNotFound) {
			return nil, ErrVenueNotFound
		}
		return nil, err
	}

	if req.EventDate.IsZero() {
		return nil, errors.New("event_date is required")
	}

	event := &model.Event{
		ID:          uuid.New(),
		Name:        title,
		Description: req.Description,
		VenueID:     req.VenueID,
		Venue:       *venue,
		Status:      model.StatusDraft,
		EventDate:   req.EventDate,
		StartDate:   req.EventDate,
		EndDate:     req.EventDate,
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	// Invalidate catalog search cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateCatalog(ctx)
	}

	return dto.ToEventResponse(event), nil
}

// GetEventByID fetches event with venue and tiers, employing cache-aside with 60s TTL
func (s *eventService) GetEventByID(ctx context.Context, id uuid.UUID) (*dto.EventDetailResponse, error) {
	cacheKey := fmt.Sprintf("%s%s", EventKeyPrefix, id.String())

	// 1. Try cache hit
	if s.cacheService != nil {
		var cached dto.EventDetailResponse
		if s.cacheService.Get(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}

	// 2. Cache miss -> query database via repository
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}

	detail := dto.ToEventDetailResponse(event)

	// 3. Populate Redis cache with 60s TTL
	if s.cacheService != nil {
		_ = s.cacheService.Set(ctx, cacheKey, detail, DefaultCacheTTL)
	}

	return detail, nil
}

// UpdateEventStatus validates lifecycle transition matrix, updates status in DB, and invalidates caches
func (s *eventService) UpdateEventStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) (*dto.EventResponse, error) {
	// Retrieve current event state
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}

	// Validate terminal state
	if event.IsTerminal() {
		return nil, model.ErrTerminalState
	}

	// Validate FSM transition
	if !event.CanTransitionTo(status) {
		return nil, model.ErrInvalidStateTransition
	}

	// Persist new status
	if err := s.eventRepo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}

	event.Status = status

	// Invalidate event detail and catalog query caches
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateEvent(ctx, id)
	}

	return dto.ToEventResponse(event), nil
}

// SearchEvents queries catalog events with filtering & pagination, cached in Redis for 60s
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

	// 1. Check cache
	if s.cacheService != nil {
		var cached dto.PaginatedEventsResponse
		if s.cacheService.Get(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}

	// 2. Query database repository
	events, total, err := s.eventRepo.List(ctx, filter, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]dto.EventResponse, len(events))
	for i := range events {
		items[i] = *dto.ToEventResponse(&events[i])
	}

	meta := dto.NewPaginationMeta(page, limit, total)
	res := &dto.PaginatedEventsResponse{
		Items: items,
		Data:  items,
		Meta:  meta,
	}

	// 3. Cache result with 60s TTL
	if s.cacheService != nil {
		_ = s.cacheService.Set(ctx, cacheKey, res, DefaultCacheTTL)
	}

	return res, nil
}

// GetEvents is an alias for SearchEvents
func (s *eventService) GetEvents(ctx context.Context, filter dto.EventFilter, page, limit int) (*dto.PaginatedEventsResponse, error) {
	return s.SearchEvents(ctx, filter, page, limit)
}

// GenerateCatalogCacheKey generates a deterministic Redis cache key for catalog search queries
func GenerateCatalogCacheKey(filter dto.EventFilter, page, limit int) string {
	var startDateStr, endDateStr string
	if filter.StartDate != nil {
		startDateStr = filter.StartDate.UTC().Format(time.RFC3339)
	}
	if filter.EndDate != nil {
		endDateStr = filter.EndDate.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("%sp=%d:l=%d:s=%s:c=%s:sd=%s:ed=%s:q=%s",
		CatalogKeyPrefix,
		page,
		limit,
		strings.ToUpper(string(filter.Status)),
		strings.ToLower(strings.TrimSpace(filter.City)),
		startDateStr,
		endDateStr,
		strings.ToLower(strings.TrimSpace(filter.Search)),
	)
}
