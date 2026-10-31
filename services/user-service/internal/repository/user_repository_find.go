// Package repository provides data persistence for user entities using GORM.
//
// File: user_repository_find.go
// Usability: Implements query lookups (FindByEmail, FindByID) and state updates (UpdatePassword, VerifyEmail).
package repository

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ? AND is_active = ?", email, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.Where("id = ? AND is_active = ?", id, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) UpdatePassword(id uuid.UUID, newPasswordHash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", newPasswordHash).Error
}

func (r *userRepository) VerifyEmail(id uuid.UUID) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("email_verified", true).Error
}
