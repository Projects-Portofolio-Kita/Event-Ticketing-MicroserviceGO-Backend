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
	ErrPricingTierNotFound = errors.New("pricing tier not found")
)

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
	if tier == nil {
		return model.ErrNilPricingTier
	}
	if err := tier.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(tier).Error
}

func (r *pricingTierRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PricingTier, error) {
	var tier model.PricingTier
	err := r.db.WithContext(ctx).First(&tier, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPricingTierNotFound
		}
		return nil, err
	}
	return &tier, nil
}

func (r *pricingTierRepository) GetByEventID(ctx context.Context, eventID uuid.UUID) ([]model.PricingTier, error) {
	var tiers []model.PricingTier
	err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("price ASC").
		Find(&tiers).Error
	if err != nil {
		return nil, err
	}
	return tiers, nil
}

func (r *pricingTierRepository) Update(ctx context.Context, tier *model.PricingTier) error {
	if tier == nil {
		return model.ErrNilPricingTier
	}
	if err := tier.Validate(); err != nil {
		return err
	}
	res := r.db.WithContext(ctx).
		Model(&model.PricingTier{}).
		Where("id = ?", tier.ID).
		Updates(map[string]interface{}{
			"tier_name":   tier.TierName,
			"price":       tier.Price,
			"currency":    tier.Currency,
			"total_seats": tier.TotalSeats,
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPricingTierNotFound
	}
	return nil
}

func (r *pricingTierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.PricingTier{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPricingTierNotFound
	}
	return nil
}
