// Package config loads runtime parameters and environment options for notification-service.
//
// File: config.go
// Usability: Declares Config struct, LoadConfig parser with fallback defaults, and Load alias.
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration variables for the notification service
type Config struct {
	ServerPort, SMTPHost, SMTPUser, SMTPPass, SMTPFrom, SMTPFromName string
	SMTPPort, WorkerCount, QueueCapacity                             int
	SMTPUseTLS                                                       bool
}

// LoadConfig loads configuration from .env file and environment variables with sensible defaults
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file loaded, reading environment variables from system")
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = getEnv("PORT", "8085")
	}
	pass := os.Getenv("SMTP_PASS")
	if pass == "" {
		pass = os.Getenv("SMTP_PASSWORD")
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = getEnv("SMTP_FROM_EMAIL", "noreply@tickid.com")
	}
	return &Config{
		ServerPort: port, SMTPHost: getEnv("SMTP_HOST", "localhost"),
		SMTPPort: getEnvInt("SMTP_PORT", 1025), SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: pass, SMTPFrom: from, SMTPFromName: getEnv("SMTP_FROM_NAME", "TickId Official"),
		SMTPUseTLS:  getEnvBool("SMTP_USE_TLS", false),
		WorkerCount: getEnvInt("WORKER_COUNT", 5), QueueCapacity: getEnvInt("QUEUE_CAPACITY", 500),
	}
}

// Load is an alias for LoadConfig
func Load() *Config {
	return LoadConfig()
}
