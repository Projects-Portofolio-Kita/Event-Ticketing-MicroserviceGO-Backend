// Package config provides runtime configuration settings for catalog-service.
//
// File: config.go
// Usability: Loads catalog-service environment variables, database URLs, and Redis addresses.
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	RedisURL    string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file loaded, reading environment variables from system")
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" { port = os.Getenv("PORT") }
	if port == "" { port = "8082" }

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" { dbURL = "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable" }

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" { redisURL = os.Getenv("REDIS_ADDR") }
	if redisURL == "" { redisURL = "localhost:6379" }

	return &Config{ServerPort: port, DatabaseURL: dbURL, RedisURL: redisURL}
}

func Load() *Config { return LoadConfig() }