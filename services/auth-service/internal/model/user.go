package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Provider records how the account was created.
type AuthProvider string

const (
	ProviderLocal  AuthProvider = "local"
	ProviderGoogle AuthProvider = "google"
)

type User struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string       `gorm:"type:varchar(100);not null" json:"name"`
	Email        string       `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string       `gorm:"type:varchar(255);not null;default:''" json:"-"`
	AvatarURL    string       `gorm:"type:varchar(512);not null;default:''" json:"avatarUrl,omitempty"`
	Provider     AuthProvider `gorm:"type:varchar(20);not null;default:'local'" json:"provider"`
	// GoogleSub is NULL for locally-registered accounts. It must stay a
	// pointer: a plain "" would collide with the unique index on the second
	// local registration and block every signup after the first.
	GoogleSub *string   `gorm:"type:varchar(255);uniqueIndex" json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// PublicUser is the safe profile shape returned to clients.
type PublicUser struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

func (u *User) Public() PublicUser {
	return PublicUser{
		ID:     u.ID.String(),
		Email:  u.Email,
		Name:   u.Name,
		Avatar: u.AvatarURL,
	}
}
