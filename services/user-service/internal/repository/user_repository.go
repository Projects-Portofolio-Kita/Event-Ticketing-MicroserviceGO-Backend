package repository

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

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

func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) Delete(id uuid.UUID) error {
	// Soft deactivate and delete
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active": false,
	}).Error
}

func (r *userRepository) UpdatePassword(id uuid.UUID, newPasswordHash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", newPasswordHash).Error
}

func (r *userRepository) VerifyEmail(id uuid.UUID) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("email_verified", true).Error
}
