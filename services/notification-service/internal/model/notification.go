package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification is one user-facing message emitted by another service.
//
// Rows are written by POST /api/v1/notifications, which booking-service and
// payment-service call through libs/common/notify.
type Notification struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	Kind      string     `gorm:"type:varchar(40);not null" json:"kind"`
	Title     string     `gorm:"type:varchar(120);not null" json:"title"`
	Body      string     `gorm:"type:text;not null" json:"body"`
	Reference string     `gorm:"type:varchar(64)" json:"reference,omitempty"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
