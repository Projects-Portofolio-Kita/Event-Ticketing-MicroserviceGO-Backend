package service

import (
	"errors"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("notification not found")
	ErrForbidden    = errors.New("Anda tidak punya akses ke notifikasi ini.")
)

// notificationStore is the slice of the repository this service depends on.
// Keeping it an interface lets tests substitute an in-memory fake.
type notificationStore interface {
	Create(notification *model.Notification) error
	ListByUser(userID uuid.UUID, limit int) ([]model.Notification, error)
	FindByID(id uuid.UUID) (*model.Notification, error)
	MarkRead(notification *model.Notification) error
}

type NotificationService struct {
	repository notificationStore
}

func NewNotificationService(repository notificationStore) *NotificationService {
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

// List reads the inbox of one user. The query's userId is only honoured when
// it matches the subject of the verified token, so a user can never list
// somebody else's notifications.
func (s *NotificationService) List(query *dto.ListNotificationsQuery, caller authz.Caller) ([]model.Notification, error) {
	if !caller.Internal && (query.UserID == "" || query.UserID != caller.UserID) {
		return nil, ErrForbidden
	}

	userID, err := uuid.Parse(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: userId", ErrInvalidInput)
	}
	return s.repository.ListByUser(userID, query.Limit)
}

// MarkRead is idempotent: reading twice keeps the original ReadAt.
func (s *NotificationService) MarkRead(id string, caller authz.Caller) (*model.Notification, error) {
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

	if !caller.Internal && caller.UserID != notification.UserID.String() {
		return nil, ErrForbidden
	}

	if err := s.repository.MarkRead(notification); err != nil {
		return nil, err
	}
	return notification, nil
}
