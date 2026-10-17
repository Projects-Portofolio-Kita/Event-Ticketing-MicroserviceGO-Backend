package handler

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all HTTP transport routes for catalog-service
func RegisterRoutes(
	r *gin.Engine,
	eventHandler *EventHandler,
	tierHandler *PricingTierHandler,
	healthHandler *HealthHandler,
) {
	if healthHandler != nil {
		r.GET("/health", healthHandler.Check)
	}

	v1 := r.Group("/api/v1")
	{
		if eventHandler != nil {
			v1.POST("/events", eventHandler.CreateEvent)
			v1.GET("/events", eventHandler.SearchEvents)
			v1.GET("/events/:id", eventHandler.GetEventByID)
			v1.PATCH("/events/:id/status", eventHandler.UpdateEventStatus)
		}

		if tierHandler != nil {
			v1.POST("/events/:id/tiers", tierHandler.AddPricingTier)
			v1.GET("/events/:id/tiers", tierHandler.GetPricingTiers)
		}
	}
}
