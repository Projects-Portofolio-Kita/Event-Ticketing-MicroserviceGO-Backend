// Package repository provides data persistence for user entities using GORM.
//
// File: user_repository.go
// Usability: Declares UserRepository interface, struct, constructor, and lifecycle write operations (Create, Update, Delete).
package repository

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines persistent operations for User entities.
type UserRepository interface {
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByID(id uuid.UUID) (*model.User, error)
	Update(user *model.User) error
	Delete(id uuid.UUID) error
	UpdatePassword(id uuid.UUID, newPasswordHash string) error
	VerifyEmail(id uuid.UUID) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository initializes a new UserRepository implementation.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) Delete(id uuid.UUID) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active": false,
	}).Error
}
