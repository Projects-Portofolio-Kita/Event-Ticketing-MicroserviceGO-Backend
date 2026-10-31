// Package repository handles persistence operations and database queries for catalog entities.
//
// File: event_repository.go
// Usability: Declares EventRepository interface, GORM implementation struct, constructor, and Create operation.
package repository

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrEventNotFound = model.ErrEventNotFound

// EventRepository defines data access methods for Event entities
type EventRepository interface {
	Create(ctx context.Context, event *model.Event) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Event, error)
	Update(ctx context.Context, event *model.Event) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) error
	List(ctx context.Context, filter EventFilter, page, limit int) ([]model.Event, int64, error)
}

type eventRepository struct {
	db *gorm.DB
}

// NewEventRepository creates a new instance of EventRepository backed by GORM
func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(ctx context.Context, event *model.Event) error {
	if event == nil {
		return model.ErrNilEvent
	}
	return r.db.WithContext(ctx).Create(event).Error
}
