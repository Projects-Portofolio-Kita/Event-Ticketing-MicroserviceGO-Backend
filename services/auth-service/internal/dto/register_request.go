package dto

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/model"
)

type RegisterRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=100"`
	Email        string `json:"email" binding:"required,email,max=255"`
	Password     string `json:"password" binding:"required,min=6,max=100"`
	CanPostEvent bool   `json:"canPostEvent"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=1,max=100"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,min=10"`
}

type GoogleRequest struct {
	IDToken string `json:"idToken" binding:"required,min=10"`
}

// UpdateProfileRequest is PATCH /api/v1/auth/me: edit the signed-in user's
// display name and avatar URL (no file upload yet — the client stores a URL
// and falls back to initials). A nil avatar means "leave it untouched", an
// empty string clears it.
type UpdateProfileRequest struct {
	Name   string  `json:"name" binding:"required,min=1,max=100"`
	Avatar *string `json:"avatar"`
}

// SessionResponse matches the web-portal contract:
// { token | accessToken, refreshToken?, user? }.
type SessionResponse struct {
	AccessToken  string           `json:"accessToken"`
	Token        string           `json:"token"`
	RefreshToken string           `json:"refreshToken"`
	User         model.PublicUser `json:"user"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
