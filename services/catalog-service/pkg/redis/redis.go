package redis

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient initializes a Redis client from REDIS_URL or fallback "localhost:6379".
// It performs a ping check with graceful fallback/logging if Redis is not running.
func NewClient(optionalURL ...string) *redis.Client {
	var targetURL string
	if len(optionalURL) > 0 && optionalURL[0] != "" {
		targetURL = optionalURL[0]
	} else {
		targetURL = os.Getenv("REDIS_URL")
		if targetURL == "" {
			targetURL = os.Getenv("REDIS_ADDR")
		}
		if targetURL == "" {
			targetURL = "localhost:6379"
		}
	}

	var opt *redis.Options
	if strings.HasPrefix(targetURL, "redis://") || strings.HasPrefix(targetURL, "rediss://") {
		var err error
		opt, err = redis.ParseURL(targetURL)
		if err != nil {
			log.Printf("[WARN] Failed to parse Redis URL (%s): %v, falling back to raw address", targetURL, err)
			opt = &redis.Options{
				Addr: targetURL,
			}
		}
	} else {
		password := os.Getenv("REDIS_PASSWORD")
		opt = &redis.Options{
			Addr:     targetURL,
			Password: password,
			DB:       0,
		}
	}

	opt.DialTimeout = 2 * time.Second
	opt.ReadTimeout = 2 * time.Second
	opt.WriteTimeout = 2 * time.Second

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
