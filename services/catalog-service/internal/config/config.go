package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration variables for the catalog service
type Config struct {
	ServerPort  string
	DatabaseURL string
	RedisURL    string
}

// LoadConfig loads configuration from .env file and environment variables with sensible defaults
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file loaded, reading environment variables from system")
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8082"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_ADDR")
	}
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	return &Config{
		ServerPort:  port,
		DatabaseURL: dbURL,
		RedisURL:    redisURL,
	}
}

// Load is an alias for LoadConfig
func Load() *Config {
	return LoadConfig()
}
