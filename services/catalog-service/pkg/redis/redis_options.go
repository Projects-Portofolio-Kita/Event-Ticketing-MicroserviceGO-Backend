// Package redis manages connections and options for Redis caching clients.
//
// File: redis_options.go
// Usability: Resolves connection URLs, authentication parameters, and network timeouts for Redis clients.
package redis

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func resolveRedisOptions(optionalURL ...string) (*redis.Options, string) {
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
			opt = &redis.Options{Addr: targetURL}
		}
	} else {
		opt = &redis.Options{Addr: targetURL, Password: os.Getenv("REDIS_PASSWORD"), DB: 0}
	}
	opt.DialTimeout, opt.ReadTimeout, opt.WriteTimeout = 2*time.Second, 2*time.Second, 2*time.Second
	return opt, targetURL
}
