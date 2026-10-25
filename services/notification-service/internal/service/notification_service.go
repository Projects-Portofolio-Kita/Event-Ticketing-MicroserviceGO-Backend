package service

import (
	"errors"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("notification not found")
)

type NotificationService struct {
	repository *repository.NotificationRepository
}

func NewNotificationService(repository *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repository: repository}
}

func (s *NotificationService) Create(req *dto.CreateNotificationRequest) (*model.Notification, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: userId", ErrInvalidInput)
	}

	notification := &model.Notification{
		UserID:    userID,
		Kind:      req.Kind,
		Title:     req.Title,
		Body:      req.Body,
		Reference: req.Reference,
	}

	if err := s.repository.Create(notification); err != nil {
		return nil, err
	}
	return notification, nil
}

func (s *NotificationService) List(query *dto.ListNotificationsQuery) ([]model.Notification, error) {
	userID, err := uuid.Parse(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: userId", ErrInvalidInput)
	}
	return s.repository.ListByUser(userID, query.Limit)
}

// MarkRead is idempotent: reading twice keeps the original ReadAt.
func (s *NotificationService) MarkRead(id string) (*model.Notification, error) {
	notificationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: id", ErrInvalidInput)
	}

	notification, err := s.repository.FindByID(notificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := s.repository.MarkRead(notification); err != nil {
		return nil, err
	}
	return notification, nil
}
