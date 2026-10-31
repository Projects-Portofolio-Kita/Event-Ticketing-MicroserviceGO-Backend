// Package redis manages connections and options for Redis caching clients.
//
// File: redis.go
// Usability: Bootstraps Redis client connections and validates server availability via ping checks.
package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient initializes a Redis client from REDIS_URL or fallback "localhost:6379".
func NewClient(optionalURL ...string) *redis.Client {
	opt, targetURL := resolveRedisOptions(optionalURL...)
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[WARN] Redis ping failed (%s): %v. Running in degraded cache mode.", targetURL, err)
	} else {
		log.Printf("[INFO] Connected to Redis at %s", targetURL)
	}
	return client
}
