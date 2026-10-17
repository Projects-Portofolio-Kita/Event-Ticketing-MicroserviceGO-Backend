package redis_test

import (
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/redis"
)

func TestNewClient_FallbackHandling(t *testing.T) {
	// Connecting to an unreachable or mock address should not panic, but return a client gracefully
	client := redis.NewClient("localhost:6379")
	if client == nil {
		t.Fatal("expected non-nil redis.Client")
	}
	defer client.Close()
}

func TestNewClient_URLParsing(t *testing.T) {
	client := redis.NewClient("redis://localhost:6379/1")
	if client == nil {
		t.Fatal("expected non-nil redis.Client with redis:// URL")
	}
	defer client.Close()
}
