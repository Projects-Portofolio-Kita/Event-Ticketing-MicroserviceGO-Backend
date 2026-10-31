// Package config handles environment variable loading and defaults for user-service.
//
// File: config.go
// Usability: Loads user-service database URLs, server ports, and JWT expiration configs.
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort                 string
	DatabaseURL                string
	JWTSecret                  string
	JWTExpirationMinutes       int
	RefreshTokenExpirationDays int
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[Config] Notice: No .env file found or failed to load, reading from environment")
	}

	serverPort := getEnv("SERVER_PORT", "8081")
	databaseURL := getEnv("DATABASE_URL", "postgres://admin:your-secret-password@localhost:5432/event_ticketing_db?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "default_jwt_secret_change_me_in_production_32char")

	jwtExpMin, err := strconv.Atoi(getEnv("JWT_EXPIRATION_MINUTES", "15"))
	if err != nil { jwtExpMin = 15 }

	refreshExpDays, err := strconv.Atoi(getEnv("REFRESH_TOKEN_EXPIRATION_DAYS", "7"))
	if err != nil { refreshExpDays = 7 }

	return &Config{
		ServerPort:                 serverPort,
		DatabaseURL:                databaseURL,
		JWTSecret:                  jwtSecret,
		JWTExpirationMinutes:       jwtExpMin,
		RefreshTokenExpirationDays: refreshExpDays,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" { return value }
	return fallback
}