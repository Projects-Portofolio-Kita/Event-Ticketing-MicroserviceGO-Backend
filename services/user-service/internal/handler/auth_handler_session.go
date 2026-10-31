// Package handler provides HTTP transport controllers for authentication endpoints.
//
// File: auth_handler_session.go
// Usability: Implements RefreshToken and Logout handlers for user session lifecycle.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RefreshToken exchanges a valid refresh token for a new access token.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}
	tokenResp, err := h.authService.RefreshToken(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(err.Error(), "ERR_REFRESH_FAILED"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Token refreshed successfully", tokenResp))
}

// Logout revokes refresh tokens and closes the user session.
func (h *AuthHandler) Logout(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Unauthorized", "ERR_UNAUTHORIZED"))
		return
	}
	userID := userIDVal.(uuid.UUID)
	var req dto.RefreshTokenRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.authService.Logout(userID, req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(err.Error(), "ERR_INTERNAL"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Logged out successfully", nil))
}
