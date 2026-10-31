// Package repository handles persistence operations and database queries for catalog entities.
//
// File: event_repository_status.go
// Usability: Implements UpdateStatus for event finite state transitions in PostgreSQL.
package repository

import (
	"context"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

func (r *eventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) error {
	res := r.db.WithContext(ctx).Model(&model.Event{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now()})
	if res.Error != nil || res.RowsAffected == 0 {
		if res.Error != nil {
			return res.Error
		}
		return ErrEventNotFound
	}
	return nil
}
