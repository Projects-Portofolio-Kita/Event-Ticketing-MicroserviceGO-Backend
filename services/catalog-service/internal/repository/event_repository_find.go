// Package repository handles persistence operations and database queries for catalog entities.
//
// File: event_repository_find.go
// Usability: Implements GetByID with preloads and general field Updates for event records.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *eventRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Event, error) {
	var event model.Event
	err := r.db.WithContext(ctx).Preload("Venue").Preload("PricingTiers").First(&event, "id = ?", id).Error
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
	res := r.db.WithContext(ctx).Model(&model.Event{}).Where("id = ?", event.ID).
		Updates(map[string]interface{}{
			"name": event.Name, "description": event.Description, "venue_id": event.VenueID,
			"event_date": event.EventDate, "start_date": event.StartDate, "end_date": event.EndDate, "updated_at": time.Now(),
		})
	if res.Error != nil || res.RowsAffected == 0 {
		if res.Error != nil {
			return res.Error
		}
		return ErrEventNotFound
	}
	return nil
}
