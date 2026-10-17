package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEventNotFound = model.ErrEventNotFound
)

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

func (r *eventRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Event, error) {
	var event model.Event
	err := r.db.WithContext(ctx).
		Preload("Venue").
		Preload("PricingTiers").
		First(&event, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) Update(ctx context.Context, event *model.Event) error {
	if event == nil {
		return model.ErrNilEvent
	}
	res := r.db.WithContext(ctx).
		Model(&model.Event{}).
		Where("id = ?", event.ID).
		Updates(map[string]interface{}{
			"name":        event.Name,
			"description": event.Description,
			"venue_id":    event.VenueID,
			"event_date":  event.EventDate,
			"start_date":  event.StartDate,
			"end_date":    event.EndDate,
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *eventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) error {
	res := r.db.WithContext(ctx).
		Model(&model.Event{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *eventRepository) List(ctx context.Context, filter EventFilter, page, limit int) ([]model.Event, int64, error) {
	buildQuery := func() *gorm.DB {
		q := r.db.WithContext(ctx).Model(&model.Event{})
		if filter.Status != "" {
			q = q.Where("events.status = ?", filter.Status)
		}
		if filter.City != "" {
			q = q.Joins("JOIN venues ON venues.id = events.venue_id").
				Where("LOWER(venues.city) LIKE LOWER(?)", "%"+filter.City+"%")
		}
		if filter.StartDate != nil {
			q = q.Where("events.event_date >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			q = q.Where("events.event_date <= ?", *filter.EndDate)
		}
		if filter.Search != "" {
			q = q.Where("LOWER(events.name) LIKE LOWER(?)", "%"+filter.Search+"%")
		}
		return q
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var events []model.Event
	err := buildQuery().
		Select("events.*").
		Preload("Venue").
		Preload("PricingTiers").
		Order("events.event_date ASC").
		Offset(offset).
		Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}
