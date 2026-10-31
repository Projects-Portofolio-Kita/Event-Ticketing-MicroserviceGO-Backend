// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: event_handler_get.go
// Usability: Implements GetEventByID endpoint retrieving full event metadata and pricing tiers.
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

// GetEventByID handles GET /api/v1/events/:id
func (h *EventHandler) GetEventByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
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
		Success: true, Message: "Event retrieved successfully", Data: res, EventDetailResponse: res,
	})
}
