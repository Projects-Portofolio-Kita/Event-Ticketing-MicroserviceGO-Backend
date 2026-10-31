// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: cache_memory_invalidation.go
// Usability: Implements memory cache purging for single event keys and prefix-matched catalog lists.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (m *inMemoryCacheService) InvalidateEvent(ctx context.Context, eventID uuid.UUID) error {
	eventKey := fmt.Sprintf("%s%s", EventKeyPrefix, eventID.String())
	tiersKey := fmt.Sprintf("%s%s:tiers", EventKeyPrefix, eventID.String())

	m.mu.Lock()
	delete(m.items, eventKey)
	delete(m.items, tiersKey)
	for k := range m.items {
		if strings.HasPrefix(k, CatalogKeyPrefix) {
			delete(m.items, k)
		}
	}
	m.mu.Unlock()
	return nil
}

func (m *inMemoryCacheService) InvalidateCatalog(ctx context.Context) error {
	m.mu.Lock()
	for k := range m.items {
		if strings.HasPrefix(k, CatalogKeyPrefix) {
			delete(m.items, k)
		}
	}
	m.mu.Unlock()
	return nil
}
