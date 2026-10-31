// Package service provides core business logic for user profiles and authentication.
//
// File: auth_service_login.go
// Usability: Implements Login to verify user credentials and issue signed JWT access/refresh token pairs.
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *authService) Login(req *dto.LoginRequest) (*dto.TokenResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, errors.New("invalid email or password")
	}
	accessTok, expIn, err := jwt.GenerateAccessToken(user, s.cfg.JWTSecret, s.cfg.JWTExpirationMinutes)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}
	refreshTok, expAt, err := jwt.GenerateRefreshToken(user, s.cfg.JWTSecret, s.cfg.RefreshTokenExpirationDays)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}
	tokenEntity := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken(refreshTok),
		ExpiresAt: expAt,
		Revoked:   false,
	}
	if err := s.tokenRepo.Create(tokenEntity); err != nil {
		return nil, errors.New("failed to store refresh token")
	}
	return &dto.TokenResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		TokenType:    "Bearer",
		ExpiresIn:    expIn,
		User:         dto.ToUserResponse(user),
	}, nil
}
