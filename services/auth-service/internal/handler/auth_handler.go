package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func errorMessage(c *gin.Context, status int, message string) {
	c.JSON(status, dto.ErrorResponse{Message: message})
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorMessage(c, http.StatusBadRequest, "Data pendaftaran tidak valid: "+err.Error())
		return
	}
	session, err := h.service.Register(req.Name, req.Email, req.Password, req.CanPostEvent)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			errorMessage(c, http.StatusConflict, "Email sudah terdaftar. Silakan masuk.")
			return
		}
		errorMessage(c, http.StatusInternalServerError, "Pendaftaran gagal. Silakan coba lagi.")
		return
	}
	c.JSON(http.StatusCreated, session)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorMessage(c, http.StatusBadRequest, "Data login tidak valid: "+err.Error())
		return
	}
	session, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			errorMessage(c, http.StatusUnauthorized, "Email atau kata sandi salah. Silakan coba lagi.")
			return
		}
		errorMessage(c, http.StatusInternalServerError, "Login gagal. Silakan coba lagi.")
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorMessage(c, http.StatusBadRequest, "Refresh token tidak valid.")
		return
	}
	session, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		errorMessage(c, http.StatusUnauthorized, "Sesi kedaluwarsa. Silakan masuk ulang.")
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /api/v1/auth/google
func (h *AuthHandler) Google(c *gin.Context) {
	var req dto.GoogleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorMessage(c, http.StatusBadRequest, "Token Google tidak valid.")
		return
	}
	session, err := h.service.LoginWithGoogle(req.IDToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGoogleNotConfigured):
			errorMessage(c, http.StatusServiceUnavailable, "Masuk dengan Google belum dikonfigurasi di server.")
		case errors.Is(err, service.ErrGoogleTokenInvalid):
			errorMessage(c, http.StatusUnauthorized, "Token Google tidak valid atau kedaluwarsa. Coba lagi.")
		default:
			errorMessage(c, http.StatusInternalServerError, "Login Google gagal. Silakan coba lagi.")
		}
		return
	}
	c.JSON(http.StatusOK, session)
}

// currentUserID reads the principal the auth middleware put on the context.
// Every handler below is mounted behind RequireAuth, so it is always set.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	raw, ok := c.Get("userID")
	if !ok {
		return uuid.Nil, false
	}
	userID, ok := raw.(uuid.UUID)
	return userID, ok
}

// GET /api/v1/auth/me (requires Bearer access token)
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		errorMessage(c, http.StatusUnauthorized, "Sesi tidak valid. Silakan masuk ulang.")
		return
	}
	user, err := h.service.Me(userID)
	if err != nil {
		errorMessage(c, http.StatusUnauthorized, "Akun tidak ditemukan. Silakan masuk ulang.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user.Public()})
}

// PATCH /api/v1/auth/me (requires Bearer access token) — edit name / avatar.
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		errorMessage(c, http.StatusUnauthorized, "Sesi tidak valid. Silakan masuk ulang.")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorMessage(c, http.StatusBadRequest, "Data profil tidak valid: "+err.Error())
		return
	}

	user, err := h.service.UpdateProfile(userID, req.Name, req.Avatar)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidProfile):
			errorMessage(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrUserNotFound):
			errorMessage(c, http.StatusUnauthorized, "Akun tidak ditemukan. Silakan masuk ulang.")
		default:
			errorMessage(c, http.StatusInternalServerError, "Memperbarui profil gagal. Silakan coba lagi.")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user.Public()})
}
