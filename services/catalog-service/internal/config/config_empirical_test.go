package config_test

import (
	"os"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/config"
)

func clearAllEnv(t *testing.T) func() {
	vars := []string{"SERVER_PORT", "PORT", "DATABASE_URL", "REDIS_URL", "REDIS_ADDR"}
	saved := make(map[string]string)
	for _, v := range vars {
		saved[v] = os.Getenv(v)
		os.Unsetenv(v)
	}

	return func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}

func TestConfig_Empirical_DefaultsExactMatch(t *testing.T) {
	restore := clearAllEnv(t)
	defer restore()

	cfg := config.LoadConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	expectedPort := "8082"
	expectedDB := "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable"
	expectedRedis := "localhost:6379"

	if cfg.ServerPort != expectedPort {
		t.Errorf("FAIL: default ServerPort expected %q, got %q", expectedPort, cfg.ServerPort)
	}
	if cfg.DatabaseURL != expectedDB {
		t.Errorf("FAIL: default DatabaseURL expected %q, got %q", expectedDB, cfg.DatabaseURL)
	}
	if cfg.RedisURL != expectedRedis {
		t.Errorf("FAIL: default RedisURL expected %q, got %q", expectedRedis, cfg.RedisURL)
	}
}

func TestConfig_Empirical_PortFallbackHierarchy(t *testing.T) {
	restore := clearAllEnv(t)
	defer restore()

	// 1. When only PORT is provided
	os.Setenv("PORT", "7070")
	cfg1 := config.LoadConfig()
	if cfg1.ServerPort != "7070" {
		t.Errorf("FAIL: expected PORT fallback 7070, got %s", cfg1.ServerPort)
	}

	// 2. When both SERVER_PORT and PORT are provided, SERVER_PORT wins
	os.Setenv("SERVER_PORT", "8888")
	cfg2 := config.LoadConfig()
	if cfg2.ServerPort != "8888" {
		t.Errorf("FAIL: expected SERVER_PORT 8888 to override PORT, got %s", cfg2.ServerPort)
	}

	// 3. When SERVER_PORT is empty string, should fall back to PORT
	os.Setenv("SERVER_PORT", "")
	cfg3 := config.LoadConfig()
	if cfg3.ServerPort != "7070" {
		t.Errorf("FAIL: expected empty SERVER_PORT to fall back to PORT 7070, got %s", cfg3.ServerPort)
	}
}

func TestConfig_Empirical_RedisFallbackHierarchy(t *testing.T) {
	restore := clearAllEnv(t)
	defer restore()

	// 1. When only REDIS_ADDR is provided
	os.Setenv("REDIS_ADDR", "redis-cluster:6380")
	cfg1 := config.LoadConfig()
	if cfg1.RedisURL != "redis-cluster:6380" {
		t.Errorf("FAIL: expected REDIS_ADDR fallback 'redis-cluster:6380', got %s", cfg1.RedisURL)
	}

	// 2. When both REDIS_URL and REDIS_ADDR are provided, REDIS_URL wins
	os.Setenv("REDIS_URL", "redis://auth@prod-redis:6379/2")
	cfg2 := config.LoadConfig()
	if cfg2.RedisURL != "redis://auth@prod-redis:6379/2" {
		t.Errorf("FAIL: expected REDIS_URL to override REDIS_ADDR, got %s", cfg2.RedisURL)
	}

	// 3. When REDIS_URL is empty string, falls back to REDIS_ADDR
	os.Setenv("REDIS_URL", "")
	cfg3 := config.LoadConfig()
	if cfg3.RedisURL != "redis-cluster:6380" {
		t.Errorf("FAIL: expected empty REDIS_URL to fall back to REDIS_ADDR, got %s", cfg3.RedisURL)
	}
}

func TestConfig_Empirical_LoadAlias(t *testing.T) {
	restore := clearAllEnv(t)
	defer restore()

	cfg1 := config.LoadConfig()
	cfg2 := config.Load()

	if *cfg1 != *cfg2 {
		t.Errorf("FAIL: Load() returned %+v but LoadConfig() returned %+v", cfg2, cfg1)
	}
}
