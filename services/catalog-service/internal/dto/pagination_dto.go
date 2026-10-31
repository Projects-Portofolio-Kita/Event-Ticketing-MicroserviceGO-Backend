// Package dto provides data transfer objects and response envelopes for the catalog service.
//
// File: pagination_dto.go
// Usability: Declares pagination request filters, metadata contracts, and PaginatedResponse wrappers.
package dto

import "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"

// EventFilter is aliased from repository.EventFilter for query parameters
type EventFilter = repository.EventFilter

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	CurrentPage  int   `json:"current_page"`
	Page         int   `json:"page,omitempty"`
	Limit        int   `json:"limit"`
	TotalItems   int64 `json:"total_items"`
	TotalRecords int64 `json:"total_records,omitempty"`
	TotalPages   int   `json:"total_pages"`
}

// PaginatedResponse wraps a slice of items along with pagination metadata
type PaginatedResponse[T any] struct {
	Items []T            `json:"items"`
	Data  []T            `json:"data,omitempty"`
	Meta  PaginationMeta `json:"meta"`
}

// PaginatedEventsResponse is a paginated response for EventResponse
type PaginatedEventsResponse = PaginatedResponse[EventResponse]

// NewPaginationMeta computes total pages and constructs PaginationMeta
func NewPaginationMeta(page, limit int, totalItems int64) PaginationMeta {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages == 0 && totalItems == 0 {
		totalPages = 0
	}
	return PaginationMeta{
		CurrentPage: page, Page: page, Limit: limit,
		TotalItems: totalItems, TotalRecords: totalItems, TotalPages: totalPages,
	}
}
