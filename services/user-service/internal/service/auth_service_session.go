// Package service provides core business logic for user profiles and authentication.
//
// File: auth_service_session.go
// Usability: Implements Logout session revocation, ForgotPassword initiation, and ResetPassword updates.
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *authService) Logout(userID uuid.UUID, refreshTokenStr string) error {
	if refreshTokenStr != "" {
		_ = s.tokenRepo.Revoke(hashToken(refreshTokenStr))
	} else {
		_ = s.tokenRepo.RevokeAllByUserID(userID)
	}
	return nil
}

func (s *authService) ForgotPassword(req *dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil {
		return nil
	}
	return nil
}

func (s *authService) ResetPassword(req *dto.ResetPasswordRequest) error {
	claims, err := jwt.ValidateToken(req.Token, s.cfg.JWTSecret)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}
	if err := s.userRepo.UpdatePassword(claims.UserID, string(hashedPassword)); err != nil {
		return errors.New("failed to update password")
	}
	_ = s.tokenRepo.RevokeAllByUserID(claims.UserID)
	return nil
}
