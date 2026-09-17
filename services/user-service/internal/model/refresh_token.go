package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	TokenHash string    `gorm:"type:varchar(255);not null;index" json:"token_hash"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null" json:"expires_at"`
	Revoked   bool      `gorm:"type:boolean;not null;default:false" json:"revoked"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
