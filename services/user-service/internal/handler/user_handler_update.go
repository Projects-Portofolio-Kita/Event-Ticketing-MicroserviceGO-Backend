// Package handler provides HTTP transport controllers for user profile endpoints.
//
// File: user_handler_update.go
// Usability: Implements UpdateProfile HTTP endpoint to update authenticated user profile details.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

// UpdateProfile handles modifying user profile details.
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}
	updated, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_UPDATE_FAILED"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Profile updated successfully", updated))
}
