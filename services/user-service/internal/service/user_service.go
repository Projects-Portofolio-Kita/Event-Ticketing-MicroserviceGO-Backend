// Package service provides core business logic for user profiles and authentication.
//
// File: user_service.go
// Usability: Defines UserService interface, struct, constructor, and user query operations (GetProfile, GetUserByID).
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/repository"
	"github.com/google/uuid"
)

// UserService defines user profile retrieval, mutation, and deactivation operations.
type UserService interface {
	GetProfile(userID uuid.UUID) (*dto.UserResponse, error)
	UpdateProfile(userID uuid.UUID, req *dto.UpdateProfileRequest) (*dto.UserResponse, error)
	GetUserByID(id uuid.UUID) (*dto.UserResponse, error)
	DeactivateAccount(userID uuid.UUID) error
}

type userService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
}

// NewUserService initializes a new UserService instance.
func NewUserService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository) UserService {
	return &userService{userRepo: userRepo, tokenRepo: tokenRepo}
}

func (s *userService) GetProfile(userID uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil { return nil, err }
	if user == nil { return nil, errors.New("user not found") }
	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *userService) GetUserByID(id uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil { return nil, err }
	if user == nil { return nil, errors.New("user not found") }
	res := dto.ToUserResponse(user)
	return &res, nil
}