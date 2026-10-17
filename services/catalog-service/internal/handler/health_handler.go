package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles service health check inquiries
type HealthHandler struct{}

// NewHealthHandler initializes a new HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check returns service health status (HTTP 200 OK)
// GET /health
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "catalog-service",
	})
}
