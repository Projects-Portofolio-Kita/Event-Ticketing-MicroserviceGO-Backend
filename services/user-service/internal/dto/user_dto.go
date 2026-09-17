package dto

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
)

type UserResponse struct {
	ID            uuid.UUID      `json:"id"`
	Email         string         `json:"email"`
	FirstName     string         `json:"first_name"`
	LastName      *string        `json:"last_name,omitempty"`
	Phone         *string        `json:"phone,omitempty"`
	AvatarURL     *string        `json:"avatar_url,omitempty"`
	Role          model.UserRole `json:"role"`
	EmailVerified bool           `json:"email_verified"`
	IsActive      bool           `json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,min=2,max=100"`
	LastName  *string `json:"last_name" binding:"omitempty,max=100"`
	Phone     *string `json:"phone" binding:"omitempty,max=20"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url"`
}

func ToUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Phone:         u.Phone,
		AvatarURL:     u.AvatarURL,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		IsActive:      u.IsActive,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
