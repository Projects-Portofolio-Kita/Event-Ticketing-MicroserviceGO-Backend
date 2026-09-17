package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_REGISTER_FAILED"))
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse("User registered successfully", user))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}

	tokenResp, err := h.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(err.Error(), "ERR_UNAUTHORIZED"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse("Login successful", tokenResp))
}

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

func (h *AuthHandler) Logout(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Unauthorized", "ERR_UNAUTHORIZED"))
		return
	}

	userID := userIDVal.(uuid.UUID)

	var req dto.RefreshTokenRequest
	_ = c.ShouldBindJSON(&req) // optional refresh token string

	if err := h.authService.Logout(userID, req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(err.Error(), "ERR_INTERNAL"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse("Logged out successfully", nil))
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}

	_ = h.authService.ForgotPassword(&req)
	c.JSON(http.StatusOK, dto.SuccessResponse("If your email is registered, a password reset link has been sent.", nil))
}

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
