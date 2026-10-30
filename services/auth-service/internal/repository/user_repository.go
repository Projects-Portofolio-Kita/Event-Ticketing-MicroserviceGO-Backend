package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrDuplicateEmail = errors.New("email already registered")
var ErrRefreshNotFound = errors.New("refresh token not found")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}

func (r *UserRepository) CreateUser(user *model.User) error {
	if err := r.db.Create(user).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ?", strings.ToLower(strings.TrimSpace(email))).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByGoogleSub(sub string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("google_sub = ?", sub).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser persists an edited profile (name / avatar URL).
func (r *UserRepository) UpdateUser(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) SaveRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	return r.db.Create(&model.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}).Error
}

func (r *UserRepository) FindValidRefreshToken(tokenHash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	if err := r.db.Where("token_hash = ? AND revoked = ? AND expires_at > ?", tokenHash, false, time.Now()).First(&rt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshNotFound
		}
		return nil, err
	}
	return &rt, nil
}

func (r *UserRepository) RevokeRefreshToken(id uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).Where("id = ?", id).Update("revoked", true).Error
}

func (r *UserRepository) RevokeAllUserTokens(userID uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error
}
