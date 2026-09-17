package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleCustomer  UserRole = "CUSTOMER"
	RoleOrganizer UserRole = "ORGANIZER"
	RoleAdmin     UserRole = "ADMIN"
)

type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email         string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash  string         `gorm:"type:varchar(255);not null" json:"-"`
	FirstName     string         `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName      *string        `gorm:"type:varchar(100)" json:"last_name,omitempty"`
	Phone         *string        `gorm:"type:varchar(20)" json:"phone,omitempty"`
	AvatarURL     *string        `gorm:"type:text" json:"avatar_url,omitempty"`
	Role          UserRole       `gorm:"type:varchar(20);not null;default:'CUSTOMER'" json:"role"`
	EmailVerified bool           `gorm:"type:boolean;not null;default:false" json:"email_verified"`
	IsActive      bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
