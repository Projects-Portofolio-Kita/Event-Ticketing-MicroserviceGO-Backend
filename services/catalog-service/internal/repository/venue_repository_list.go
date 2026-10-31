// Package repository handles persistence operations and database queries for catalog entities.
//
// File: venue_repository_list.go
// Usability: Implements List, Count, and idempotent SeedVenues queries for venue datasets.
package repository

import (
	"context"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"gorm.io/gorm/clause"
)

func (r *venueRepository) List(ctx context.Context) ([]model.Venue, error) {
	var venues []model.Venue
	err := r.db.WithContext(ctx).Order("name ASC").Find(&venues).Error
	if err != nil {
		return nil, err
	}
	return venues, nil
}

func (r *venueRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Venue{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *venueRepository) SeedVenues(ctx context.Context, venues []model.Venue) error {
	if len(venues) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&venues).Error
}
