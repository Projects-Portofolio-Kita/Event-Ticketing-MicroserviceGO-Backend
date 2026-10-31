// Package repository handles persistence operations and database queries for catalog entities.
//
// File: venue_repository.go
// Usability: Declares VenueRepository interface, struct, constructor, and GetByID primary key retrieval.
package repository

import (
	"context"
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrVenueNotFound = errors.New("venue not found")
)

// VenueRepository defines data access methods for Venue entities
type VenueRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Venue, error)
	List(ctx context.Context) ([]model.Venue, error)
	Count(ctx context.Context) (int64, error)
	SeedVenues(ctx context.Context, venues []model.Venue) error
}

type venueRepository struct {
	db *gorm.DB
}

// NewVenueRepository creates a new instance of VenueRepository backed by GORM
func NewVenueRepository(db *gorm.DB) VenueRepository {
	return &venueRepository{db: db}
}

func (r *venueRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Venue, error) {
	var venue model.Venue
	err := r.db.WithContext(ctx).First(&venue, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVenueNotFound
		}
		return nil, err
	}
	return &venue, nil
}
