// Package service provides core business logic for user profiles and authentication.
//
// File: auth_service_register.go
// Usability: Implements Register to validate uniqueness, hash passwords with bcrypt, and persist new user accounts.
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *authService) Register(req *dto.RegisterRequest) (*dto.UserResponse, error) {
	existing, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}
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
