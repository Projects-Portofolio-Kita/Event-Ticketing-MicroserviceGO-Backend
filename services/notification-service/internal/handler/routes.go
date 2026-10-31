package handler

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and configures the Gin engine with all routes and standard middleware.
func SetupRouter(notifHandler *NotificationHandler, healthHandler *HealthHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Health check endpoint
	router.GET("/health", healthHandler.Health)

	// API v1 notification routes
	v1 := router.Group("/api/v1/notifications")
	{
		v1.POST("/ticket-confirmation", notifHandler.SendTicketConfirmation)
		v1.POST("/payment-failed", notifHandler.SendPaymentFailed)
		v1.POST("/send-custom", notifHandler.SendCustomEmail)
	}

	return router
}
