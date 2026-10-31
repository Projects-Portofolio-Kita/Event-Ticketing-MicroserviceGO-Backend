// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: cache_redis_invalidation.go
// Usability: Implements Redis key eviction for single events and cursor-scanned catalog list purges.
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

func (s *redisCacheService) InvalidateEvent(ctx context.Context, eventID uuid.UUID) error {
	if s.client == nil {
		return nil
	}
	eventKey := fmt.Sprintf("%s%s", EventKeyPrefix, eventID.String())
	tiersKey := fmt.Sprintf("%s%s:tiers", EventKeyPrefix, eventID.String())
	if err := s.client.Del(ctx, eventKey, tiersKey).Err(); err != nil {
		log.Printf("[WARN] Redis Del failed for event keys (%s, %s): %v", eventKey, tiersKey, err)
	}
	return s.InvalidateCatalog(ctx)
}

func (s *redisCacheService) InvalidateCatalog(ctx context.Context) error {
	if s.client == nil {
		return nil
	}
	var cursor uint64
	matchPattern := CatalogKeyPrefix + "*"
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, matchPattern, 100).Result()
		if err != nil {
			log.Printf("[WARN] Redis Scan failed during InvalidateCatalog for %q: %v", matchPattern, err)
			return nil
		}
		if len(keys) > 0 {
			if delErr := s.client.Del(ctx, keys...).Err(); delErr != nil {
				log.Printf("[WARN] Redis Del failed during InvalidateCatalog: %v", delErr)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
