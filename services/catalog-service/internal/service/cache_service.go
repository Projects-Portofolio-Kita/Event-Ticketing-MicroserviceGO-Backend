package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultCacheTTL is the standard 60-second TTL for catalog and event caches
	DefaultCacheTTL = 60 * time.Second

	// EventKeyPrefix is the prefix for individual event cache entries
	EventKeyPrefix = "catalog:event:"

	// CatalogKeyPrefix is the prefix for list/search catalog queries
	CatalogKeyPrefix = "catalog:events:"
)

// CacheService defines the cache-aside contract for catalog operations
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) bool
	Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error
	InvalidateEvent(ctx context.Context, eventID uuid.UUID) error
	InvalidateCatalog(ctx context.Context) error
}

// redisCacheService implements CacheService backed by a Redis client
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
	return &redisCacheService{
		client:     client,
		defaultTTL: ttl,
	}
}

// Get retrieves a key from Redis and unmarshals JSON into dest. Returns true on cache hit.
func (s *redisCacheService) Get(ctx context.Context, key string, dest interface{}) bool {
	if s.client == nil {
		return false
	}

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("[WARN] Redis Get failed for key %q: %v", key, err)
		}
		return false
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		log.Printf("[WARN] Failed to unmarshal cached JSON for key %q: %v", key, err)
		return false
	}

	return true
}

// Set serializes val to JSON and caches it in Redis with the given TTL.
// If Redis fails, logs warning and returns nil for transparent fallback.
func (s *redisCacheService) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	if s.client == nil {
		return nil
	}

	if ttl <= 0 {
		ttl = s.defaultTTL
	}

	data, err := json.Marshal(val)
	if err != nil {
		log.Printf("[WARN] Failed to JSON marshal value for cache key %q: %v", key, err)
		return err
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("[WARN] Redis Set failed for key %q: %v", key, err)
		return nil // Transparent fallback: do not fail caller
	}

	return nil
}

// InvalidateEvent evicts the event detail cache, its tiers cache, and all catalog search keys
func (s *redisCacheService) InvalidateEvent(ctx context.Context, eventID uuid.UUID) error {
	if s.client == nil {
		return nil
	}

	eventKey := fmt.Sprintf("%s%s", EventKeyPrefix, eventID.String())
	tiersKey := fmt.Sprintf("%s%s:tiers", EventKeyPrefix, eventID.String())

	if err := s.client.Del(ctx, eventKey, tiersKey).Err(); err != nil {
		log.Printf("[WARN] Redis Del failed for event keys (%s, %s): %v", eventKey, tiersKey, err)
	}

	// Always evict catalog list cache on event mutations
	return s.InvalidateCatalog(ctx)
}

// InvalidateCatalog purges all catalog search and list keys matching "catalog:events:*"
func (s *redisCacheService) InvalidateCatalog(ctx context.Context) error {
	if s.client == nil {
		return nil
	}

	var cursor uint64
	matchPattern := CatalogKeyPrefix + "*"

	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, matchPattern, 100).Result()
		if err != nil {
			log.Printf("[WARN] Redis Scan failed during InvalidateCatalog for pattern %q: %v", matchPattern, err)
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

// ---------------------------------------------------------------------------
// In-Memory Cache Implementation (for offline / unit-testing / standalone fallback)
// ---------------------------------------------------------------------------

type memoryItem struct {
	data      []byte
	expiresAt time.Time
}

type inMemoryCacheService struct {
	mu         sync.RWMutex
	items      map[string]memoryItem
	defaultTTL time.Duration
}

// NewInMemoryCacheService creates an in-memory CacheService for tests or zero-dependency fallback
func NewInMemoryCacheService(defaultTTL ...time.Duration) CacheService {
	ttl := DefaultCacheTTL
	if len(defaultTTL) > 0 && defaultTTL[0] > 0 {
		ttl = defaultTTL[0]
	}
	return &inMemoryCacheService{
		items:      make(map[string]memoryItem),
		defaultTTL: ttl,
	}
}

func (m *inMemoryCacheService) Get(ctx context.Context, key string, dest interface{}) bool {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()

	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			m.mu.Lock()
			delete(m.items, key)
			m.mu.Unlock()
		}
		return false
	}

	if err := json.Unmarshal(item.data, dest); err != nil {
		return false
	}
	return true
}

func (m *inMemoryCacheService) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = m.defaultTTL
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.items[key] = memoryItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	m.mu.Unlock()
	return nil
}

func (m *inMemoryCacheService) InvalidateEvent(ctx context.Context, eventID uuid.UUID) error {
	eventKey := fmt.Sprintf("%s%s", EventKeyPrefix, eventID.String())
	tiersKey := fmt.Sprintf("%s%s:tiers", EventKeyPrefix, eventID.String())

	m.mu.Lock()
	delete(m.items, eventKey)
	delete(m.items, tiersKey)

	// Invalidate catalog search query keys
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
