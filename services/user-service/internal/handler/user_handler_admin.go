// Package handler provides HTTP transport controllers for user profile endpoints.
//
// File: user_handler_admin.go
// Usability: Implements administrative and account deactivation endpoints (GetUserByID, DeactivateAccount).
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUserByID retrieves a user record by administrative lookup.
func (h *UserHandler) GetUserByID(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("Invalid user ID format", "ERR_INVALID_UUID"))
		return
	}
	user, err := h.userService.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse(err.Error(), "ERR_USER_NOT_FOUND"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("User found", user))
}

// DeactivateAccount soft deletes the user account.
func (h *UserHandler) DeactivateAccount(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	if err := h.userService.DeactivateAccount(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(err.Error(), "ERR_INTERNAL"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Account deactivated successfully", nil))
}
