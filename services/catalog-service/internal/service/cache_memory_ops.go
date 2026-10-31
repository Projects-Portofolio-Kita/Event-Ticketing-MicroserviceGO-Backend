// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: cache_memory_ops.go
// Usability: Implements thread-safe Get and Set operations with expiration checks for inMemoryCacheService.
package service

import (
	"context"
	"encoding/json"
	"time"
)

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
	return json.Unmarshal(item.data, dest) == nil
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
