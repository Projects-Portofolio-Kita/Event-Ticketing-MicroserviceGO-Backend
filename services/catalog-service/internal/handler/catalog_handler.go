package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
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
	events, err := h.service.ListEvents()
	if err != nil {
		httperr.FromError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *CatalogHandler) GetSeatMap(c *gin.Context) {
	categories, err := h.service.GetSeatMap(c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			httperr.FromError(c, http.StatusBadRequest, err)
		case errors.Is(err, service.ErrEventNotFound):
			httperr.NotFound(c, err.Error())
		default:
			httperr.FromError(c, http.StatusInternalServerError, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
