// Package repository handles persistence operations and database queries for catalog entities.
//
// File: pricing_tier_repository.go
// Usability: Declares PricingTierRepository interface, struct, constructor, and Create / GetByID operations.
package repository

import (
	"context"
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrPricingTierNotFound = errors.New("pricing tier not found")

// PricingTierRepository defines data access methods for PricingTier entities
type PricingTierRepository interface {
	Create(ctx context.Context, tier *model.PricingTier) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.PricingTier, error)
	GetByEventID(ctx context.Context, eventID uuid.UUID) ([]model.PricingTier, error)
	Update(ctx context.Context, tier *model.PricingTier) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pricingTierRepository struct {
	db *gorm.DB
}

// NewPricingTierRepository creates a new instance of PricingTierRepository backed by GORM
func NewPricingTierRepository(db *gorm.DB) PricingTierRepository {
	return &pricingTierRepository{db: db}
}

func (r *pricingTierRepository) Create(ctx context.Context, tier *model.PricingTier) error {
	if tier == nil { return model.ErrNilPricingTier }
	if err := tier.Validate(); err != nil { return err }
	return r.db.WithContext(ctx).Create(tier).Error
}

func (r *pricingTierRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PricingTier, error) {
	var tier model.PricingTier
	err := r.db.WithContext(ctx).First(&tier, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrPricingTierNotFound }
		return nil, err
	}
	return &tier, nil
}