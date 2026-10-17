package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PricingTierHandler handles HTTP transport for event pricing tiers
type PricingTierHandler struct {
	tierService service.PricingTierService
}

// NewPricingTierHandler creates a new PricingTierHandler instance
func NewPricingTierHandler(tierService service.PricingTierService) *PricingTierHandler {
	return &PricingTierHandler{tierService: tierService}
}

type pricingTierResponseEnvelope struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message,omitempty"`
	Data    *dto.PricingTierResponse `json:"data,omitempty"`
	*dto.PricingTierResponse
}

type pricingTiersResponseEnvelope struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message,omitempty"`
	Data    []dto.PricingTierResponse `json:"data"`
	Tiers   []dto.PricingTierResponse `json:"tiers,omitempty"`
}

// AddPricingTier handles POST /api/v1/events/:id/tiers
// Adds a new pricing tier to an event; rejects with 409 Conflict if event is ON_SALE or SOLD_OUT
func (h *PricingTierHandler) AddPricingTier(c *gin.Context) {
	idParam := c.Param("id")
	eventID, err := uuid.Parse(idParam)
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
		if errors.Is(err, model.ErrPricingTierLocked) {
			c.JSON(http.StatusConflict, dto.ErrorResponse[any](err.Error()))
			return
		}
		if errors.Is(err, model.ErrEventNotFound) || errors.Is(err, repository.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse[any]("event not found"))
			return
		}
		if errors.Is(err, model.ErrInvalidTierPrice) || errors.Is(err, model.ErrInvalidTierSeats) || errors.Is(err, model.ErrInvalidTierName) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse[any](err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse[any](err.Error()))
		return
	}

	c.JSON(http.StatusCreated, pricingTierResponseEnvelope{
		Success:             true,
		Message:             "Pricing tier added successfully",
		Data:                res,
		PricingTierResponse: res,
	})
}

// AddTier is an alias for AddPricingTier
func (h *PricingTierHandler) AddTier(c *gin.Context) {
	h.AddPricingTier(c)
}

// GetPricingTiers handles GET /api/v1/events/:id/tiers
// Returns all pricing tiers defined for the event
func (h *PricingTierHandler) GetPricingTiers(c *gin.Context) {
	idParam := c.Param("id")
	eventID, err := uuid.Parse(idParam)
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
		Success: true,
		Message: "Pricing tiers retrieved successfully",
		Data:    res,
		Tiers:   res,
	})
}

// GetTiers is an alias for GetPricingTiers
func (h *PricingTierHandler) GetTiers(c *gin.Context) {
	h.GetPricingTiers(c)
}
