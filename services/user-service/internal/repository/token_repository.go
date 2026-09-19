package repository

import (
	"errors"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenRepository interface {
	Create(token *model.RefreshToken) error
	FindByHash(tokenHash string) (*model.RefreshToken, error)
	Revoke(tokenHash string) error
	RevokeAllByUserID(userID uuid.UUID) error
}

type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *tokenRepository) FindByHash(tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.Where("token_hash = ? AND revoked = ? AND expires_at > ?", tokenHash, false, time.Now()).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) Revoke(tokenHash string) error {
	return r.db.Model(&model.RefreshToken{}).Where("token_hash = ?", tokenHash).Update("revoked", true).Error
}

func (r *tokenRepository) RevokeAllByUserID(userID uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error
}
