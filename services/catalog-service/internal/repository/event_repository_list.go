// Package repository handles persistence operations and database queries for catalog entities.
//
// File: event_repository_list.go
// Usability: Implements paginated event search filtering across dates, cities, and statuses.
package repository

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
)

func (r *eventRepository) List(ctx context.Context, filter EventFilter, page, limit int) ([]model.Event, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Event{})
	if filter.Status != "" {
		q = q.Where("events.status = ?", filter.Status)
	}
	if filter.City != "" {
		q = q.Joins("JOIN venues ON venues.id = events.venue_id").Where("LOWER(venues.city) LIKE LOWER(?)", "%"+filter.City+"%")
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
	var total int64
	if err := q.Count(&total).Error; err != nil {
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
	var events []model.Event
	err := q.Select("events.*").Preload("Venue").Preload("PricingTiers").
		Order("events.event_date ASC").Offset((page - 1) * limit).Limit(limit).Find(&events).Error
	return events, total, err
}
