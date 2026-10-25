package repository

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CatalogRepository struct {
	db *gorm.DB
}

func NewCatalogRepository(db *gorm.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) FindEvents() ([]model.Event, error) {
	var events []model.Event

	err := r.db.Order("created_at ASC").Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (r *CatalogRepository) FindByID(id uuid.UUID) (*model.Event, error) {
	var event model.Event

	err := r.db.First(&event, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (r *CatalogRepository) FindCategoriesByEvent(eventID uuid.UUID) ([]model.SeatCategory, error) {
	var categories []model.SeatCategory

	err := r.db.
		Where("event_id = ?", eventID).
		Order("sort_order ASC").
		Find(&categories).Error

	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *CatalogRepository) FindSeatsByEvent(eventID uuid.UUID) ([]model.Seat, error) {
	var seats []model.Seat

	err := r.db.
		Where("event_id = ?", eventID).
		Order("category_id ASC, sort_order ASC").
		Find(&seats).Error

	if err != nil {
		return nil, err
	}

	return seats, nil
}
