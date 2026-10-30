// Package cors provides the single CORS policy shared by every service.
//
// The portal and the desktop app never talk to a service directly in dev
// (Vite/nginx proxy everything), but the API is also reachable on its own
// port during development, so permissive CORS is applied uniformly here
// instead of being copy-pasted into each service's internal/middleware.
package cors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// New returns a middleware that answers preflight requests and opens the
// API to any origin. Identical behaviour for every service, so a change here
// propagates to all of them at once.
func New() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
