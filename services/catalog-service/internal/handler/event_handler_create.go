// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: event_handler_create.go
// Usability: Implements CreateEvent endpoint to register new events in DRAFT state.
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
)

// CreateEvent handles POST /api/v1/events
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
		Success: true, Message: "Event created successfully", Data: res, EventResponse: res,
	})
}
