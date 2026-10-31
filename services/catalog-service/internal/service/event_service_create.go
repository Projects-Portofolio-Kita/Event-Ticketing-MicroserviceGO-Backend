// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: event_service_create.go
// Usability: Implements CreateEvent with venue existence verification and catalog cache eviction.
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
		ID: uuid.New(), Name: title, Description: req.Description,
		VenueID: req.VenueID, Venue: *venue, Status: model.StatusDraft,
		EventDate: req.EventDate, StartDate: req.EventDate, EndDate: req.EventDate,
	}
	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateCatalog(ctx)
	}
	return dto.ToEventResponse(event), nil
}
