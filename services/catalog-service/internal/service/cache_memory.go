// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: cache_memory.go
// Usability: Declares in-memory thread-safe fallback cache structures and constructors.
package service

import (
	"sync"
	"time"
)

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
