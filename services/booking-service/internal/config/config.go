package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort      string
	DatabaseURL     string
	RedisAddr       string
	LockTTL         time.Duration
	SweeperInterval time.Duration
}

func LoadConfig() *Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8083"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_ADDR")
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	ttlMin := 5
	if val := os.Getenv("LOCK_TTL_MINUTES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			ttlMin = parsed
		}
	}

	sweepSec := 30
	if val := os.Getenv("SWEEPER_INTERVAL_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			sweepSec = parsed
		}
	}

	return &Config{
		ServerPort:      port,
		DatabaseURL:     dbURL,
		RedisAddr:       redisAddr,
		LockTTL:         time.Duration(ttlMin) * time.Minute,
		SweeperInterval: time.Duration(sweepSec) * time.Second,
	}
}
