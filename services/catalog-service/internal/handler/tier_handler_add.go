// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: tier_handler_add.go
// Usability: Implements AddPricingTier and AddTier HTTP endpoint actions with status mapping.
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

// AddPricingTier handles POST /api/v1/events/:id/tiers
func (h *PricingTierHandler) AddPricingTier(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any]("invalid event ID format"))
		return
	}
	var req dto.CreatePricingTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any](err.Error()))
		return
	}
	res, err := h.tierService.AddPricingTier(c.Request.Context(), eventID, req)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrPricingTierLocked):
			c.JSON(http.StatusConflict, dto.ErrorResponse[any](err.Error()))
		case errors.Is(err, model.ErrEventNotFound) || errors.Is(err, repository.ErrEventNotFound):
			c.JSON(http.StatusNotFound, dto.ErrorResponse[any]("event not found"))
		case errors.Is(err, model.ErrInvalidTierPrice) || errors.Is(err, model.ErrInvalidTierSeats) || errors.Is(err, model.ErrInvalidTierName):
			c.JSON(http.StatusBadRequest, dto.ErrorResponse[any](err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		}
		return
	}
	c.JSON(http.StatusCreated, pricingTierResponseEnvelope{Success: true, Message: "Pricing tier added successfully", Data: res, PricingTierResponse: res})
}

// AddTier is an alias for AddPricingTier
func (h *PricingTierHandler) AddTier(c *gin.Context) { h.AddPricingTier(c) }