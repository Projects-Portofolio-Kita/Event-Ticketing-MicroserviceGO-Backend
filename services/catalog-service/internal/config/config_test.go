package config_test

import (
	"os"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/config"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Temporarily clear environment variables to test defaults
	origPort := os.Getenv("SERVER_PORT")
	origDB := os.Getenv("DATABASE_URL")
	origRedis := os.Getenv("REDIS_URL")

	defer func() {
		os.Setenv("SERVER_PORT", origPort)
		os.Setenv("DATABASE_URL", origDB)
		os.Setenv("REDIS_URL", origRedis)
	}()

	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("PORT")
	os.Unsetenv("REDIS_ADDR")

	cfg := config.LoadConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.ServerPort == "" {
		t.Error("expected non-empty ServerPort")
	}
	if cfg.DatabaseURL == "" {
		t.Error("expected non-empty DatabaseURL")
	}
	if cfg.RedisURL == "" {
		t.Error("expected non-empty RedisURL")
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	origPort := os.Getenv("SERVER_PORT")
	origDB := os.Getenv("DATABASE_URL")
	origRedis := os.Getenv("REDIS_URL")

	defer func() {
		os.Setenv("SERVER_PORT", origPort)
		os.Setenv("DATABASE_URL", origDB)
		os.Setenv("REDIS_URL", origRedis)
	}()

	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("DATABASE_URL", "postgres://user:pass@testhost:5432/testdb")
	os.Setenv("REDIS_URL", "redis://testhost:6379")

	cfg := config.LoadConfig()
	if cfg.ServerPort != "9999" {
		t.Errorf("expected ServerPort '9999', got '%s'", cfg.ServerPort)
	}
	if cfg.DatabaseURL != "postgres://user:pass@testhost:5432/testdb" {
		t.Errorf("expected custom DatabaseURL, got '%s'", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://testhost:6379" {
		t.Errorf("expected custom RedisURL, got '%s'", cfg.RedisURL)
	}
}
