// Package service provides core business logic for user profiles and authentication.
//
// File: auth_service_refresh.go
// Usability: Implements RefreshToken with cryptographic verification and single-use token rotation.
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
)

func (s *authService) RefreshToken(req *dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	claims, err := jwt.ValidateToken(req.RefreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	oldHash := hashToken(req.RefreshToken)
	tokenRecord, err := s.tokenRepo.FindByHash(oldHash)
	if err != nil || tokenRecord == nil || tokenRecord.Revoked {
		return nil, errors.New("refresh token is revoked or expired")
	}
	_ = s.tokenRepo.Revoke(oldHash)
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user not found or inactive")
	}
	accessTok, expIn, err := jwt.GenerateAccessToken(user, s.cfg.JWTSecret, s.cfg.JWTExpirationMinutes)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}
	refreshTok, expAt, err := jwt.GenerateRefreshToken(user, s.cfg.JWTSecret, s.cfg.RefreshTokenExpirationDays)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}
	newTokenEntity := &model.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: hashToken(refreshTok), ExpiresAt: expAt, Revoked: false}
	if err := s.tokenRepo.Create(newTokenEntity); err != nil {
		return nil, errors.New("failed to store refresh token")
	}
	return &dto.TokenResponse{AccessToken: accessTok, RefreshToken: refreshTok, TokenType: "Bearer", ExpiresIn: expIn, User: dto.ToUserResponse(user)}, nil
}
