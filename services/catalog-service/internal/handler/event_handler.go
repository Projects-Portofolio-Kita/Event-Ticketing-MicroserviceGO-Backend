package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EventHandler handles HTTP requests related to Event domain
type EventHandler struct {
	eventService service.EventService
}

// NewEventHandler creates a new EventHandler instance
func NewEventHandler(eventService service.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

// Envelope wrappers enabling dual-compatibility:
// 1) deserialization into dto.BaseResponse[T] (reading .Data)
// 2) deserialization directly into the payload DTO (reading top-level fields)
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

// CreateEvent handles POST /api/v1/events
// Creates a new event in DRAFT status referencing a valid venue
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any](err.Error()))
		return
	}

	res, err := h.eventService.CreateEvent(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrVenueNotFound) || errors.Is(err, repository.ErrVenueNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse[any]("venue not found"))
			return
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse[any](err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}

	c.JSON(http.StatusCreated, eventResponseEnvelope{
		Success:       true,
		Message:       "Event created successfully",
		Data:          res,
		EventResponse: res,
	})
}

// GetEventByID handles GET /api/v1/events/:id
// Returns full event metadata with nested venue and pricing tiers
func (h *EventHandler) GetEventByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any]("invalid event ID format"))
		return
	}

	res, err := h.eventService.GetEventByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrEventNotFound) || errors.Is(err, repository.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse[any]("event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}

	c.JSON(http.StatusOK, eventDetailResponseEnvelope{
		Success:             true,
		Message:             "Event retrieved successfully",
		Data:                res,
		EventDetailResponse: res,
	})
}

// UpdateEventStatus handles PATCH /api/v1/events/:id/status
// Enforces FSM state transition matrix and rejects invalid transitions with HTTP 422
func (h *EventHandler) UpdateEventStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any]("invalid event ID format"))
		return
	}

	var req dto.UpdateEventStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any](err.Error()))
		return
	}

	res, err := h.eventService.UpdateEventStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, model.ErrEventNotFound) || errors.Is(err, repository.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse[any]("event not found"))
			return
		}
		if errors.Is(err, model.ErrInvalidStateTransition) || errors.Is(err, model.ErrTerminalState) {
			c.JSON(http.StatusUnprocessableEntity, dto.ErrorResponse[any](err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}

	c.JSON(http.StatusOK, eventResponseEnvelope{
		Success:       true,
		Message:       "Event status updated successfully",
		Data:          res,
		EventResponse: res,
	})
}

// SearchEvents handles GET /api/v1/events
// Queries catalog with filtering (status, city, date range) and pagination
func (h *EventHandler) SearchEvents(c *gin.Context) {
	status := c.Query("status")
	city := c.Query("city")
	search := c.Query("search")
	startDate := parseQueryDate(c.Query("start_date"))
	endDate := parseQueryDate(c.Query("end_date"))

	page := 1
	if pStr := c.DefaultQuery("page", "1"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if lStr := c.DefaultQuery("limit", "10"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	filter := dto.EventFilter{
		Status:    model.EventStatus(status),
		City:      city,
		Search:    search,
		StartDate: startDate,
		EndDate:   endDate,
	}

	res, err := h.eventService.SearchEvents(c.Request.Context(), filter, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}

	c.JSON(http.StatusOK, paginatedEventsResponseEnvelope{
		Success:                 true,
		Message:                 "Events retrieved successfully",
		Data:                    res,
		PaginatedEventsResponse: res,
	})
}

// GetEvents is an alias for SearchEvents to satisfy both convention styles
func (h *EventHandler) GetEvents(c *gin.Context) {
	h.SearchEvents(c)
}

// parseQueryDate parses date strings across multiple common RFC and ISO formats
func parseQueryDate(str string) *time.Time {
	str = strings.TrimSpace(str)
	if str == "" {
		return nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, str); err == nil {
			return &t
		}
	}
	return nil
}
