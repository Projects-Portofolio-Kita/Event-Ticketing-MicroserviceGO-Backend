// Package repository handles persistence operations and database queries for catalog entities.
//
// File: pricing_tier_repository_manage.go
// Usability: Implements GetByEventID, Update, and Delete operations for event pricing tiers.
package repository

import (
	"context"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

func (r *pricingTierRepository) GetByEventID(ctx context.Context, eventID uuid.UUID) ([]model.PricingTier, error) {
	var tiers []model.PricingTier
	err := r.db.WithContext(ctx).Where("event_id = ?", eventID).Order("price ASC").Find(&tiers).Error
	if err != nil { return nil, err }
	return tiers, nil
}

func (r *pricingTierRepository) Update(ctx context.Context, tier *model.PricingTier) error {
	if tier == nil { return model.ErrNilPricingTier }
	if err := tier.Validate(); err != nil { return err }
	res := r.db.WithContext(ctx).Model(&model.PricingTier{}).Where("id = ?", tier.ID).
		Updates(map[string]interface{}{
			"tier_name": tier.TierName, "price": tier.Price,
			"currency": tier.Currency, "total_seats": tier.TotalSeats,
			"updated_at": time.Now(),
		})
	if res.Error != nil { return res.Error }
	if res.RowsAffected == 0 { return ErrPricingTierNotFound }
	return nil
}

func (r *pricingTierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.PricingTier{})
	if res.Error != nil { return res.Error }
	if res.RowsAffected == 0 { return ErrPricingTierNotFound }
	return nil
}