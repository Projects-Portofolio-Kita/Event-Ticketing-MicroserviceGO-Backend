// Package handler provides HTTP transport controllers for user profile endpoints.
//
// File: user_handler.go
// Usability: Defines UserHandler struct, constructor, context user ID extraction, and GetProfile endpoint.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles user profile HTTP requests.
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler initializes a new UserHandler.
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func getUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Unauthorized", "ERR_UNAUTHORIZED"))
		return uuid.Nil, false
	}
	return val.(uuid.UUID), true
}

// GetProfile returns the authenticated user's profile.
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse(err.Error(), "ERR_USER_NOT_FOUND"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Profile retrieved successfully", profile))
}
