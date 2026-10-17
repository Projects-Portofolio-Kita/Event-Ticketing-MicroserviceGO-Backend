package dto

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

// CreatePricingTierRequest represents request payload to create a new pricing tier
type CreatePricingTierRequest struct {
	TierName   string  `json:"tier_name" binding:"required,min=2,max=100"`
	Price      float64 `json:"price" binding:"required,gte=0"`
	Currency   string  `json:"currency" binding:"omitempty"`
	TotalSeats int     `json:"total_seats" binding:"required,gt=0"`
}

// PricingTierResponse represents public response for a pricing tier
type PricingTierResponse struct {
	ID         uuid.UUID `json:"id"`
	EventID    uuid.UUID `json:"event_id"`
	TierName   string    `json:"tier_name"`
	Price      float64   `json:"price"`
	Currency   string    `json:"currency"`
	TotalSeats int       `json:"total_seats"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ToPricingTierResponse maps model.PricingTier to PricingTierResponse
func ToPricingTierResponse(pt *model.PricingTier) *PricingTierResponse {
	if pt == nil {
		return nil
	}
	currency := pt.Currency
	if currency == "" {
		currency = "IDR"
	}
	return &PricingTierResponse{
		ID:         pt.ID,
		EventID:    pt.EventID,
		TierName:   pt.TierName,
		Price:      pt.Price,
		Currency:   currency,
		TotalSeats: pt.TotalSeats,
		CreatedAt:  pt.CreatedAt,
		UpdatedAt:  pt.UpdatedAt,
	}
}
