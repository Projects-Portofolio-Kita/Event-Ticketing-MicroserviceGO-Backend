// Package handler provides HTTP transport controllers for authentication endpoints.
//
// File: auth_handler.go
// Usability: Defines AuthHandler struct, constructor, Register, and Login HTTP endpoints.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles user authentication HTTP requests.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler initializes a new AuthHandler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register creates a new user account.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST")); return
	}
	user, err := h.authService.Register(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_REGISTER_FAILED")); return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse("User registered successfully", user))
}

// Login validates credentials and issues JWT access/refresh tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST")); return
	}
	tokenResp, err := h.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(err.Error(), "ERR_UNAUTHORIZED")); return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse("Login successful", tokenResp))
}