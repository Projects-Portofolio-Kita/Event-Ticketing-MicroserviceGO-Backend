package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/repository"
	"github.com/google/uuid"
)

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

func NewUserService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository) UserService {
	return &userService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
	}
}

func (s *userService) GetProfile(userID uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *userService) UpdateProfile(userID uuid.UUID, req *dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = req.LastName
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *userService) GetUserByID(id uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *userService) DeactivateAccount(userID uuid.UUID) error {
	if err := s.userRepo.Delete(userID); err != nil {
		return err
	}
	_ = s.tokenRepo.RevokeAllByUserID(userID)
	return nil
}
