// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: cache_service.go
// Usability: Declares CacheService interface contract, cache key prefixes, TTLs, and Redis cache constructor.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultCacheTTL  = 60 * time.Second
	EventKeyPrefix   = "catalog:event:"
	CatalogKeyPrefix = "catalog:events:"
)

// CacheService defines the cache-aside contract for catalog operations
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) bool
	Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error
	InvalidateEvent(ctx context.Context, eventID uuid.UUID) error
	InvalidateCatalog(ctx context.Context) error
}

type redisCacheService struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewRedisCacheService creates a new Redis-backed CacheService with fallback resilience
func NewRedisCacheService(client *redis.Client, defaultTTL ...time.Duration) CacheService {
	ttl := DefaultCacheTTL
	if len(defaultTTL) > 0 && defaultTTL[0] > 0 {
		ttl = defaultTTL[0]
	}
	return &redisCacheService{client: client, defaultTTL: ttl}
}
