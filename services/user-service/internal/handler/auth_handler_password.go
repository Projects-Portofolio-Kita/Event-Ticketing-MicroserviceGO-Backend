// Package handler provides HTTP transport controllers for authentication endpoints.
//
// File: auth_handler_password.go
// Usability: Implements ForgotPassword and ResetPassword handlers for credential recovery.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

// ForgotPassword handles initiating password reset workflows.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}
	_ = h.authService.ForgotPassword(&req)
	c.JSON(http.StatusOK, dto.SuccessResponse("If your email is registered, a password reset link has been sent.", nil))
}

// ResetPassword applies a new password given a verified reset token.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}
	if err := h.authService.ResetPassword(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_RESET_PASSWORD_FAILED"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Password reset successful. Please login with your new password.", nil))
}
