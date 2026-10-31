// Package service provides core business logic for user profiles and authentication.
//
// File: auth_service.go
// Usability: Declares AuthService interface, authService struct, constructor, and token hashing helper.
package service

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/repository"
	"github.com/google/uuid"
)

// AuthService manages user registration, login, token refresh, and password recovery.
type AuthService interface {
	Register(req *dto.RegisterRequest) (*dto.UserResponse, error)
	Login(req *dto.LoginRequest) (*dto.TokenResponse, error)
	RefreshToken(req *dto.RefreshTokenRequest) (*dto.TokenResponse, error)
	Logout(userID uuid.UUID, refreshTokenStr string) error
	ForgotPassword(req *dto.ForgotPasswordRequest) error
	ResetPassword(req *dto.ResetPasswordRequest) error
}

type authService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	cfg       *config.Config
}

// NewAuthService creates a new AuthService instance with repositories and configuration.
func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}
}

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
