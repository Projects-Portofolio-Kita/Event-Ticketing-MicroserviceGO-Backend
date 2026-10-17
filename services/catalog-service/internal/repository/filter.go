package repository

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
)

// EventFilter defines the query parameters for filtering and searching events
type EventFilter struct {
	Status    model.EventStatus `json:"status" form:"status"`
	City      string            `json:"city" form:"city"`
	StartDate *time.Time        `json:"start_date" form:"start_date"`
	EndDate   *time.Time        `json:"end_date" form:"end_date"`
	Search    string            `json:"search" form:"search"`
}
