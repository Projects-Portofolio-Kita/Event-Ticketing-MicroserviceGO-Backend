// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: event_service_status.go
// Usability: Implements UpdateEventStatus validating FSM transitions and evicting relevant caches.
package service

import (
	"context"
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
)

func (s *eventService) UpdateEventStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) (*dto.EventResponse, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, model.ErrEventNotFound
		}
		return nil, err
	}
	if event.IsTerminal() {
		return nil, model.ErrTerminalState
	}
	if !event.CanTransitionTo(status) {
		return nil, model.ErrInvalidStateTransition
	}
	if err := s.eventRepo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	event.Status = status
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateEvent(ctx, id)
	}
	return dto.ToEventResponse(event), nil
}
