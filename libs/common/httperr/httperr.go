// Package httperr is the shared JSON error envelope.
//
// Every service answers failures with {"message": "..."}; the web portal
// (web-portal/src/services/*.ts) reads exactly this key, so the shape is
// defined once here and cannot drift between services.
package httperr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type body struct {
	Message string `json:"message"`
}

// JSON writes the shared error envelope with the given HTTP status.
func JSON(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, body{Message: message})
}

// FromError writes the shared envelope using err.Error() as the message.
func FromError(c *gin.Context, status int, err error) {
	JSON(c, status, err.Error())
}

// BadRequest is shorthand for HTTP 400.
func BadRequest(c *gin.Context, message string) { JSON(c, http.StatusBadRequest, message) }

// Unauthorized is shorthand for HTTP 401.
func Unauthorized(c *gin.Context, message string) { JSON(c, http.StatusUnauthorized, message) }

// NotFound is shorthand for HTTP 404.
func NotFound(c *gin.Context, message string) { JSON(c, http.StatusNotFound, message) }

// Internal is shorthand for HTTP 500.
func Internal(c *gin.Context, message string) { JSON(c, http.StatusInternalServerError, message) }
