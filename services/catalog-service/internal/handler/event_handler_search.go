// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: event_handler_search.go
// Usability: Implements SearchEvents and GetEvents HTTP endpoints for querying catalog events with filters.
package handler

import (
	"net/http"
	"strconv"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/gin-gonic/gin"
)

// SearchEvents handles GET /api/v1/events
func (h *EventHandler) SearchEvents(c *gin.Context) {
	page, limit := 1, 10
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil && l > 0 {
		limit = l
	}
	filter := dto.EventFilter{
		Status:    model.EventStatus(c.Query("status")),
		City:      c.Query("city"),
		Search:    c.Query("search"),
		StartDate: parseQueryDate(c.Query("start_date")),
		EndDate:   parseQueryDate(c.Query("end_date")),
	}
	res, err := h.eventService.SearchEvents(c.Request.Context(), filter, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}
	c.JSON(http.StatusOK, paginatedEventsResponseEnvelope{
		Success: true, Message: "Events retrieved successfully", Data: res, PaginatedEventsResponse: res,
	})
}

// GetEvents is an alias for SearchEvents
func (h *EventHandler) GetEvents(c *gin.Context) {
	h.SearchEvents(c)
}
