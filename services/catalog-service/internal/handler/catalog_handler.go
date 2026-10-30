package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
)

type CatalogHandler struct {
	service *service.CatalogService
}

func NewCatalogHandler(service *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: service}
}

func (h *CatalogHandler) ListEvents(c *gin.Context) {
	var filter dto.EventFilter
	// Unknown query keys are ignored on purpose: clients may add tracking
	// params without breaking discovery.
	_ = c.ShouldBindQuery(&filter)

	events, total, err := h.service.ListEvents(filter)
	if err != nil {
		httperr.FromError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": total})
}

// ListCities feeds the location picker dropdown (GET /events/cities).
func (h *CatalogHandler) ListCities(c *gin.Context) {
	cities, err := h.service.ListCities()
	if err != nil {
		httperr.FromError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"cities": cities})
}

// GetEvent returns one event's discovery/detail payload.
func (h *CatalogHandler) GetEvent(c *gin.Context) {
	event, err := h.service.GetEvent(c.Param("id"))
	if err != nil {
		writeCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetTicketTypes returns a TIER event's tiers with remaining stock.
func (h *CatalogHandler) GetTicketTypes(c *gin.Context) {
	response, err := h.service.GetTicketTypes(c.Param("id"))
	if err != nil {
		writeCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetTierQuotas is the internal stock feed for booking-service; it requires
// the shared X-Service-Token (wired in main.go).
func (h *CatalogHandler) GetTierQuotas(c *gin.Context) {
	response, err := h.service.GetTierQuotas(c.Param("id"))
	if err != nil {
		writeCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// writeCatalogError maps service errors onto the shared HTTP envelope.
func writeCatalogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		httperr.FromError(c, http.StatusBadRequest, err)
	case errors.Is(err, service.ErrEventNotFound):
		httperr.NotFound(c, err.Error())
	default:
		httperr.FromError(c, http.StatusInternalServerError, err)
	}
}

func (h *CatalogHandler) GetSeatMap(c *gin.Context) {
	categories, err := h.service.GetSeatMap(c.Param("id"))
	if err != nil {
		writeCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// CreateEvent posts a new event for the signed-in user (the identity comes
// from the access token, never from the body).
func (h *CatalogHandler) CreateEvent(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	event, err := h.service.CreateEvent(&req, authz.CallerFrom(c))
	if err != nil {
		writeCatalogError(c, err)
		return
	}

	// Answer with the same discovery shape as GET /events/:id so clients do
	// not have to special-case the create response. The event was written in
	// one transaction, so the re-read cannot miss it.
	if enriched, err := h.service.GetEvent(event.ID.String()); err == nil {
		c.JSON(http.StatusCreated, enriched)
		return
	}
	c.JSON(http.StatusCreated, event)
}
