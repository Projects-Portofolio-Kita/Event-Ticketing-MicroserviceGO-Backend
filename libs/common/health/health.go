// Package health exposes the one health endpoint every service publishes.
//
// Load balancers, the gateway and the portal's landing status section all
// rely on this exact payload, so it lives in one place rather than being
// re-declared per service.
package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the body returned by GET /health on every service.
type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// Handler returns the GET /health handler for the named service.
// The name is informational and must match the compose service name.
func Handler(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{Status: "ok", Service: service})
	}
}

// Register mounts GET /health on the given router.
func Register(router gin.IRoutes, service string) {
	router.GET("/health", Handler(service))
}
