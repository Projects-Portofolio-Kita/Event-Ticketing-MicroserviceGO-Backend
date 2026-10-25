package service

import (
	"errors"
	"fmt"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidInput  = errors.New("invalid input")
)

// eventStore is the consumer-side slice of repository.CatalogRepository the
// service actually uses. Depending on this narrow interface keeps the service
// testable with an in-memory fake; *repository.CatalogRepository satisfies it.
type eventStore interface {
	FindEvents() ([]model.Event, error)
	FindByID(id uuid.UUID) (*model.Event, error)
	FindCategoriesByEvent(eventID uuid.UUID) ([]model.SeatCategory, error)
	FindSeatsByEvent(eventID uuid.UUID) ([]model.Seat, error)
}

type CatalogService struct {
	repository eventStore
	booking    client.BookingClient
}

func NewCatalogService(repository eventStore, booking client.BookingClient) *CatalogService {
	return &CatalogService{
		repository: repository,
		booking:    booking,
	}
}

func (s *CatalogService) ListEvents() ([]dto.EventResponse, error) {
	events, err := s.repository.FindEvents()
	if err != nil {
		return nil, err
	}

	result := make([]dto.EventResponse, 0, len(events))
	for _, event := range events {
		result = append(result, dto.EventResponse{
			ID:    event.ID.String(),
			Code:  event.Code,
			Name:  event.Name,
			Venue: event.Venue,
		})
	}

	return result, nil
}

func (s *CatalogService) GetSeatMap(eventID string) ([]dto.CategoryResponse, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

	if _, err := s.repository.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	categories, err := s.repository.FindCategoriesByEvent(id)
	if err != nil {
		return nil, err
	}

	seats, err := s.repository.FindSeatsByEvent(id)
	if err != nil {
		return nil, err
	}

	// Seat availability is owned by booking-service; degrade to "available" when it is down.
	statuses := map[string]string{}
	if liveStatuses, err := s.booking.GetSeatStatus(eventID); err == nil {
		for _, status := range liveStatuses {
			statuses[status.SeatID] = status.Status
		}
	}

	seatsByCategory := map[uuid.UUID][]model.Seat{}
	for _, seat := range seats {
		seatsByCategory[seat.CategoryID] = append(seatsByCategory[seat.CategoryID], seat)
	}

	result := make([]dto.CategoryResponse, 0, len(categories))
	for _, category := range categories {
		rows := buildRows(seatsByCategory[category.ID], category, statuses)

		result = append(result, dto.CategoryResponse{
			ID:       category.Code,
			Name:     category.Name,
			Subtitle: category.Subtitle,
			DotColor: category.DotColor,
			Price:    category.Price,
			Rows:     rows,
		})
	}

	return result, nil
}

func buildRows(categorySeats []model.Seat, category model.SeatCategory, statuses map[string]string) []dto.RowResponse {
	rowOrder := []string{}
	rowsByName := map[string]*dto.RowResponse{}

	for _, seat := range categorySeats {
		row, exists := rowsByName[seat.RowName]
		if !exists {
			row = &dto.RowResponse{
				RowName:    seat.RowName,
				LeftSeats:  []dto.SeatResponse{},
				RightSeats: []dto.SeatResponse{},
			}
			rowsByName[seat.RowName] = row
			rowOrder = append(rowOrder, seat.RowName)
		}

		item := dto.SeatResponse{
			ID:       seat.Label,
			SeatID:   seat.ID.String(),
			Row:      seat.RowName,
			Number:   seat.Number,
			Status:   resolveStatus(statuses[seat.ID.String()]),
			Category: category.Name,
			Price:    category.Price,
		}

		if seat.Side == "right" {
			row.RightSeats = append(row.RightSeats, item)
		} else {
			row.LeftSeats = append(row.LeftSeats, item)
		}
	}

	rows := make([]dto.RowResponse, 0, len(rowOrder))
	for _, rowName := range rowOrder {
		rows = append(rows, *rowsByName[rowName])
	}

	return rows
}

func resolveStatus(bookingStatus string) string {
	switch bookingStatus {
	case "CONFIRMED":
		return "sold"
	case "LOCKED":
		return "locked"
	default:
		return "available"
	}
}
