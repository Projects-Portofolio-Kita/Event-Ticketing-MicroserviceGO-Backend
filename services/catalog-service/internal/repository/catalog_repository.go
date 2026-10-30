package repository

import (
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
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

// FindEvents lists events for the discovery view: free-text search, location,
// category and sale-mode filters, newest first, paginated.
func (r *CatalogRepository) FindEvents(filter dto.EventFilter) ([]model.Event, error) {
	var events []model.Event

	q := r.db.Model(&model.Event{})

	if term := strings.TrimSpace(filter.Q); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		q = q.Where(
			"LOWER(name) LIKE ? OR LOWER(venue) LIKE ? OR LOWER(description) LIKE ?",
			like, like, like,
		)
	}
	if city := strings.TrimSpace(filter.City); city != "" {
		q = q.Where("LOWER(city) = LOWER(?)", city)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		q = q.Where("LOWER(category) = LOWER(?)", category)
	}
	if mode := strings.TrimSpace(filter.Mode); mode != "" {
		q = q.Where("UPPER(sale_mode) = UPPER(?)", mode)
	}
	if creator := strings.TrimSpace(filter.CreatedBy); creator != "" {
		q = q.Where("created_by = ?", creator)
	}

	page, limit := filter.Normalized()

	err := q.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

// FindEventCount mirrors FindEvents' filters without pagination; the handler
// reports it alongside the page so clients can render "x hasil".
func (r *CatalogRepository) FindEventCount(filter dto.EventFilter) (int64, error) {
	q := r.db.Model(&model.Event{})

	if term := strings.TrimSpace(filter.Q); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		q = q.Where(
			"LOWER(name) LIKE ? OR LOWER(venue) LIKE ? OR LOWER(description) LIKE ?",
			like, like, like,
		)
	}
	if city := strings.TrimSpace(filter.City); city != "" {
		q = q.Where("LOWER(city) = LOWER(?)", city)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		q = q.Where("LOWER(category) = LOWER(?)", category)
	}
	if mode := strings.TrimSpace(filter.Mode); mode != "" {
		q = q.Where("UPPER(sale_mode) = UPPER(?)", mode)
	}
	if creator := strings.TrimSpace(filter.CreatedBy); creator != "" {
		q = q.Where("created_by = ?", creator)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// FindDistinctCities feeds the location picker: every city that has at least
// one event, with how many, ordered alphabetically.
func (r *CatalogRepository) FindDistinctCities() ([]dto.CityEntry, error) {
	var cities []dto.CityEntry

	err := r.db.Model(&model.Event{}).
		Select("city, COUNT(*) AS count").
		Group("city").
		Order("city ASC").
		Scan(&cities).Error
	if err != nil {
		return nil, err
	}

	return cities, nil
}

// FindCategoriesByEvents loads the categories of several events in one query
// so the list endpoint can compute minPrice without N+1 lookups.
func (r *CatalogRepository) FindCategoriesByEvents(eventIDs []uuid.UUID) ([]model.SeatCategory, error) {
	if len(eventIDs) == 0 {
		return []model.SeatCategory{}, nil
	}

	var categories []model.SeatCategory
	err := r.db.
		Where("event_id IN ?", eventIDs).
		Order("event_id ASC, sort_order ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}

	return categories, nil
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

// CodeExists reports whether some event already uses the given code.
// Comparison is case-insensitive: codes are slugged to lower case, so a
// legacy uppercase seed code must still block "coldplay".
func (r *CatalogRepository) CodeExists(code string) (bool, error) {
	var count int64

	err := r.db.Model(&model.Event{}).Where("LOWER(code) = LOWER(?)", code).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateEventWithSeatmap persists an event, its categories and its generated
// seats in a single transaction, so a half-posted event can never exist.
func (r *CatalogRepository) CreateEventWithSeatmap(event *model.Event, categories []model.SeatCategory, seats []model.Seat) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(event).Error; err != nil {
			return err
		}
		if len(categories) > 0 {
			if err := tx.Create(&categories).Error; err != nil {
				return err
			}
		}
		if len(seats) > 0 {
			if err := tx.CreateInBatches(&seats, 250).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
