// Package service implements domain business logic, caching orchestration, and workflows.
//
// File: cache_redis_ops.go
// Usability: Implements JSON unmarshal cache Get and set-with-TTL operations for redisCacheService.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func (s *redisCacheService) Get(ctx context.Context, key string, dest interface{}) bool {
	if s.client == nil { return false }
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

func (s *redisCacheService) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	if s.client == nil { return nil }
	if ttl <= 0 { ttl = s.defaultTTL }
	data, err := json.Marshal(val)
	if err != nil {
		log.Printf("[WARN] Failed to JSON marshal value for cache key %q: %v", key, err)
		return err
	}
	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("[WARN] Redis Set failed for key %q: %v", key, err)
	}
	return nil
}