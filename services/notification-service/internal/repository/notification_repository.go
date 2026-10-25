package repository

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultLimit = 20

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

// ListByUser returns a user's notifications, newest first.
func (r *NotificationRepository) ListByUser(userID uuid.UUID, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	var notifications []model.Notification
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

func (r *NotificationRepository) FindByID(id uuid.UUID) (*model.Notification, error) {
	var notification model.Notification
	if err := r.db.Where("id = ?", id).First(&notification).Error; err != nil {
		return nil, err
	}
	return &notification, nil
}

func (r *NotificationRepository) MarkRead(notification *model.Notification) error {
	if notification.ReadAt == nil {
		now := time.Now()
		notification.ReadAt = &now
	}
	return r.db.Save(notification).Error
}
