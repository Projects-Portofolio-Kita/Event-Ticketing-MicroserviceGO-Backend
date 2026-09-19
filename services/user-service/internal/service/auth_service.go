package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

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

func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}
}

func (s *authService) Register(req *dto.RegisterRequest) (*dto.UserResponse, error) {
	// Check if user already exists
	existing, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	role := model.RoleCustomer
	if req.Role != nil {
		role = *req.Role
	}

	newUser := &model.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Phone:         req.Phone,
		Role:          role,
		EmailVerified: false,
		IsActive:      true,
	}

	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	res := dto.ToUserResponse(newUser)
	return &res, nil
}

func (s *authService) Login(req *dto.LoginRequest) (*dto.TokenResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate Access Token
	accessToken, expiresIn, err := jwt.GenerateAccessToken(user, s.cfg.JWTSecret, s.cfg.JWTExpirationMinutes)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	// Generate Refresh Token
	refreshTokenRaw, expiresAt, err := jwt.GenerateRefreshToken(user, s.cfg.JWTSecret, s.cfg.RefreshTokenExpirationDays)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	// Store hash of refresh token
	tokenHash := hashToken(refreshTokenRaw)
	tokenEntity := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	if err := s.tokenRepo.Create(tokenEntity); err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User:         dto.ToUserResponse(user),
	}, nil
}

func (s *authService) RefreshToken(req *dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	// Validate JWT structure
	claims, err := jwt.ValidateToken(req.RefreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	tokenHash := hashToken(req.RefreshToken)
	tokenRecord, err := s.tokenRepo.FindByHash(tokenHash)
	if err != nil {
		return nil, err
	}
	if tokenRecord == nil || tokenRecord.Revoked {
		return nil, errors.New("refresh token is revoked or expired")
	}

	// Revoke current token (rotation)
	_ = s.tokenRepo.Revoke(tokenHash)

	// Fetch fresh user
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user not found or inactive")
	}

	// Issue new tokens
	accessToken, expiresIn, err := jwt.GenerateAccessToken(user, s.cfg.JWTSecret, s.cfg.JWTExpirationMinutes)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	newRefreshTokenRaw, expiresAt, err := jwt.GenerateRefreshToken(user, s.cfg.JWTSecret, s.cfg.RefreshTokenExpirationDays)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	newTokenHash := hashToken(newRefreshTokenRaw)
	newTokenEntity := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	if err := s.tokenRepo.Create(newTokenEntity); err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenRaw,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User:         dto.ToUserResponse(user),
	}, nil
}

func (s *authService) Logout(userID uuid.UUID, refreshTokenStr string) error {
	if refreshTokenStr != "" {
		tokenHash := hashToken(refreshTokenStr)
		_ = s.tokenRepo.Revoke(tokenHash)
	} else {
		_ = s.tokenRepo.RevokeAllByUserID(userID)
	}
	return nil
}

func (s *authService) ForgotPassword(req *dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil {
		// Do not leak email existence, return success anyway
		return nil
	}
	// In production, publish forgot_password event or generate reset token
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

	// Revoke all existing sessions for safety
	_ = s.tokenRepo.RevokeAllByUserID(claims.UserID)
	return nil
}

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
