// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: event_service_cache_key.go
// Usability: Generates deterministic Redis cache key strings based on query filter parameters.
package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
)

// GenerateCatalogCacheKey generates a deterministic Redis cache key for catalog search queries
func GenerateCatalogCacheKey(filter dto.EventFilter, page, limit int) string {
	var startDateStr, endDateStr string
	if filter.StartDate != nil {
		startDateStr = filter.StartDate.UTC().Format(time.RFC3339)
	}
	if filter.EndDate != nil {
		endDateStr = filter.EndDate.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("%sp=%d:l=%d:s=%s:c=%s:sd=%s:ed=%s:q=%s",
		CatalogKeyPrefix,
		page,
		limit,
		strings.ToUpper(string(filter.Status)),
		strings.ToLower(strings.TrimSpace(filter.City)),
		startDateStr,
		endDateStr,
		strings.ToLower(strings.TrimSpace(filter.Search)),
	)
}
