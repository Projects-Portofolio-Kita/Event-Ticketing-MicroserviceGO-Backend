package config_test

import (
	"os"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/config"
)

func clearEnv(keys []string) func() {
	orig := make(map[string]string)
	for _, k := range keys {
		orig[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for k, v := range orig {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	cleanup := clearEnv([]string{
		"SERVER_PORT", "PORT",
		"SMTP_HOST", "SMTP_PORT",
		"SMTP_USER", "SMTP_PASS", "SMTP_PASSWORD",
		"SMTP_FROM", "SMTP_FROM_EMAIL", "SMTP_FROM_NAME",
		"SMTP_USE_TLS",
		"WORKER_COUNT", "QUEUE_CAPACITY",
	})
	defer cleanup()

	cfg := config.LoadConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.ServerPort != "8085" {
		t.Errorf("expected ServerPort '8085', got '%s'", cfg.ServerPort)
	}
	if cfg.SMTPHost != "localhost" {
		t.Errorf("expected SMTPHost 'localhost', got '%s'", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 1025 {
		t.Errorf("expected SMTPPort 1025, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPUser != "" {
		t.Errorf("expected empty SMTPUser, got '%s'", cfg.SMTPUser)
	}
	if cfg.SMTPPass != "" {
		t.Errorf("expected empty SMTPPass, got '%s'", cfg.SMTPPass)
	}
	if cfg.SMTPFrom != "noreply@tickid.com" {
		t.Errorf("expected SMTPFrom 'noreply@tickid.com', got '%s'", cfg.SMTPFrom)
	}
	if cfg.SMTPFromName != "TickId Official" {
		t.Errorf("expected SMTPFromName 'TickId Official', got '%s'", cfg.SMTPFromName)
	}
	if cfg.SMTPUseTLS != false {
		t.Errorf("expected SMTPUseTLS false, got %v", cfg.SMTPUseTLS)
	}
	if cfg.WorkerCount != 5 {
		t.Errorf("expected WorkerCount 5, got %d", cfg.WorkerCount)
	}
	if cfg.QueueCapacity != 500 {
		t.Errorf("expected QueueCapacity 500, got %d", cfg.QueueCapacity)
	}

	// Test Load() alias
	cfgAlias := config.Load()
	if cfgAlias == nil || cfgAlias.ServerPort != "8085" {
		t.Errorf("Load() alias returned unexpected result: %+v", cfgAlias)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	cleanup := clearEnv([]string{
		"SERVER_PORT", "PORT",
		"SMTP_HOST", "SMTP_PORT",
		"SMTP_USER", "SMTP_PASS", "SMTP_PASSWORD",
		"SMTP_FROM", "SMTP_FROM_EMAIL", "SMTP_FROM_NAME",
		"SMTP_USE_TLS",
		"WORKER_COUNT", "QUEUE_CAPACITY",
	})
	defer cleanup()

	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SMTP_HOST", "smtp.gmail.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USER", "admin@tickid.com")
	os.Setenv("SMTP_PASS", "app-password-1234")
	os.Setenv("SMTP_FROM", "tickets@tickid.com")
	os.Setenv("SMTP_FROM_NAME", "TickId Dispatcher")
	os.Setenv("SMTP_USE_TLS", "true")
	os.Setenv("WORKER_COUNT", "12")
	os.Setenv("QUEUE_CAPACITY", "1000")

	cfg := config.LoadConfig()

	if cfg.ServerPort != "9090" {
		t.Errorf("expected ServerPort '9090', got '%s'", cfg.ServerPort)
	}
	if cfg.SMTPHost != "smtp.gmail.com" {
		t.Errorf("expected SMTPHost 'smtp.gmail.com', got '%s'", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 587 {
		t.Errorf("expected SMTPPort 587, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPUser != "admin@tickid.com" {
		t.Errorf("expected SMTPUser 'admin@tickid.com', got '%s'", cfg.SMTPUser)
	}
	if cfg.SMTPPass != "app-password-1234" {
		t.Errorf("expected SMTPPass 'app-password-1234', got '%s'", cfg.SMTPPass)
	}
	if cfg.SMTPFrom != "tickets@tickid.com" {
		t.Errorf("expected SMTPFrom 'tickets@tickid.com', got '%s'", cfg.SMTPFrom)
	}
	if cfg.SMTPFromName != "TickId Dispatcher" {
		t.Errorf("expected SMTPFromName 'TickId Dispatcher', got '%s'", cfg.SMTPFromName)
	}
	if !cfg.SMTPUseTLS {
		t.Errorf("expected SMTPUseTLS true, got %v", cfg.SMTPUseTLS)
	}
	if cfg.WorkerCount != 12 {
		t.Errorf("expected WorkerCount 12, got %d", cfg.WorkerCount)
	}
	if cfg.QueueCapacity != 1000 {
		t.Errorf("expected QueueCapacity 1000, got %d", cfg.QueueCapacity)
	}
}

func TestLoadConfig_FallbacksAndEdgeCases(t *testing.T) {
	cleanup := clearEnv([]string{
		"SERVER_PORT", "PORT",
		"SMTP_HOST", "SMTP_PORT",
		"SMTP_USER", "SMTP_PASS", "SMTP_PASSWORD",
		"SMTP_FROM", "SMTP_FROM_EMAIL", "SMTP_FROM_NAME",
		"SMTP_USE_TLS",
		"WORKER_COUNT", "QUEUE_CAPACITY",
	})
	defer cleanup()

	// Fallback to PORT, SMTP_PASSWORD, SMTP_FROM_EMAIL
	os.Setenv("PORT", "8888")
	os.Setenv("SMTP_PASSWORD", "fallback-secret")
	os.Setenv("SMTP_FROM_EMAIL", "fallback@tickid.com")
	os.Setenv("SMTP_PORT", "not-a-number")
	os.Setenv("WORKER_COUNT", "invalid-int")
	os.Setenv("QUEUE_CAPACITY", "invalid-cap")
	os.Setenv("SMTP_USE_TLS", "1")

	cfg := config.LoadConfig()

	if cfg.ServerPort != "8888" {
		t.Errorf("expected ServerPort fallback to '8888', got '%s'", cfg.ServerPort)
	}
	if cfg.SMTPPass != "fallback-secret" {
		t.Errorf("expected SMTPPass fallback to 'fallback-secret', got '%s'", cfg.SMTPPass)
	}
	if cfg.SMTPFrom != "fallback@tickid.com" {
		t.Errorf("expected SMTPFrom fallback to 'fallback@tickid.com', got '%s'", cfg.SMTPFrom)
	}
	if cfg.SMTPPort != 1025 {
		t.Errorf("expected SMTPPort fallback to 1025, got %d", cfg.SMTPPort)
	}
	if cfg.WorkerCount != 5 {
		t.Errorf("expected WorkerCount fallback to 5, got %d", cfg.WorkerCount)
	}
	if cfg.QueueCapacity != 500 {
		t.Errorf("expected QueueCapacity fallback to 500, got %d", cfg.QueueCapacity)
	}
	if !cfg.SMTPUseTLS {
		t.Errorf("expected SMTPUseTLS true for '1', got %v", cfg.SMTPUseTLS)
	}
}
