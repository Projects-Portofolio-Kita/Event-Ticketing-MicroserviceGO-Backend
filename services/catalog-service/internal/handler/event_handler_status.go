// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: event_handler_status.go
// Usability: Implements UpdateEventStatus endpoint with 422 Unprocessable Entity for invalid FSM transitions.
package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UpdateEventStatus handles PATCH /api/v1/events/:id/status
func (h *EventHandler) UpdateEventStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
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
		Success: true, Message: "Event status updated successfully", Data: res, EventResponse: res,
	})
}
