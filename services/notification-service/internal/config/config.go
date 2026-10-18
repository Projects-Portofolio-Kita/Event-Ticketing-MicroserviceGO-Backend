package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration variables for the notification service
type Config struct {
	ServerPort    string
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPass      string
	SMTPFrom      string
	SMTPFromName  string
	SMTPUseTLS    bool
	WorkerCount   int
	QueueCapacity int
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
		port = "8085"
	}

	smtpHost := getEnv("SMTP_HOST", "localhost")
	smtpPort := getEnvInt("SMTP_PORT", 1025)

	smtpUser := os.Getenv("SMTP_USER")

	smtpPass := os.Getenv("SMTP_PASS")
	if smtpPass == "" {
		smtpPass = os.Getenv("SMTP_PASSWORD")
	}

	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = os.Getenv("SMTP_FROM_EMAIL")
	}
	if smtpFrom == "" {
		smtpFrom = "noreply@tickid.com"
	}

	smtpFromName := getEnv("SMTP_FROM_NAME", "TickId Official")
	smtpUseTLS := getEnvBool("SMTP_USE_TLS", false)

	workerCount := getEnvInt("WORKER_COUNT", 5)
	queueCapacity := getEnvInt("QUEUE_CAPACITY", 500)

	return &Config{
		ServerPort:    port,
		SMTPHost:      smtpHost,
		SMTPPort:      smtpPort,
		SMTPUser:      smtpUser,
		SMTPPass:      smtpPass,
		SMTPFrom:      smtpFrom,
		SMTPFromName:  smtpFromName,
		SMTPUseTLS:    smtpUseTLS,
		WorkerCount:   workerCount,
		QueueCapacity: queueCapacity,
	}
}

// Load is an alias for LoadConfig
func Load() *Config {
	return LoadConfig()
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvBool(key string, defaultVal bool) bool {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return defaultVal
	}
	valStr = strings.TrimSpace(strings.ToLower(valStr))
	if valStr == "true" || valStr == "1" || valStr == "yes" {
		return true
	}
	if valStr == "false" || valStr == "0" || valStr == "no" {
		return false
	}
	return defaultVal
}
