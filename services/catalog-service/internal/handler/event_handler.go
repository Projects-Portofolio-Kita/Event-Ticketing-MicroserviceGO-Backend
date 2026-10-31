// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: event_handler.go
// Usability: Declares EventHandler struct, constructor, and JSON compatibility response envelopes.
package handler

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
)

// EventHandler handles HTTP requests related to Event domain
type EventHandler struct {
	eventService service.EventService
}

// NewEventHandler creates a new EventHandler instance
func NewEventHandler(eventService service.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

type eventResponseEnvelope struct {
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
	Data    *dto.EventResponse `json:"data,omitempty"`
	*dto.EventResponse
}

type eventDetailResponseEnvelope struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message,omitempty"`
	Data    *dto.EventDetailResponse `json:"data,omitempty"`
	*dto.EventDetailResponse
}

type paginatedEventsResponseEnvelope struct {
	Success bool                         `json:"success"`
	Message string                       `json:"message,omitempty"`
	Data    *dto.PaginatedEventsResponse `json:"data,omitempty"`
	*dto.PaginatedEventsResponse
}
