// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: tier_handler_get.go
// Usability: Implements GetPricingTiers and GetTiers HTTP endpoint actions to list event tiers.
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

// GetPricingTiers handles GET /api/v1/events/:id/tiers
func (h *PricingTierHandler) GetPricingTiers(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse[any]("invalid event ID format"))
		return
	}
	res, err := h.tierService.GetPricingTiers(c.Request.Context(), eventID)
	if err != nil {
		if errors.Is(err, model.ErrEventNotFound) || errors.Is(err, repository.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse[any]("event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}
	c.JSON(http.StatusOK, pricingTiersResponseEnvelope{
		Success: true, Message: "Pricing tiers retrieved successfully",
		Data: res, Tiers: res,
	})
}

// GetTiers is an alias for GetPricingTiers
func (h *PricingTierHandler) GetTiers(c *gin.Context) {
	h.GetPricingTiers(c)
}
