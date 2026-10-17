package redis_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/redis"
)

func TestRedis_Empirical_OfflineRefusal(t *testing.T) {
	// 1. Connection refused should log gracefully and return non-nil client without panic
	client := redis.NewClient("localhost:6379")
	if client == nil {
		t.Fatal("expected non-nil redis.Client even when Redis server is offline")
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err == nil {
		t.Log("Note: Redis was actually reachable at localhost:6379")
	} else {
		t.Logf("Observed expected offline error: %v", err)
	}

	// Verify executing commands does not panic
	getErr := client.Get(ctx, "nonexistent_key").Err()
	if getErr == nil {
		t.Log("Note: Key was fetched")
	} else {
		t.Logf("Observed expected command error on offline Redis: %v", getErr)
	}
}

func TestRedis_Empirical_TimeoutUnreachableIP(t *testing.T) {
	// 2. Unreachable blackhole IP (RFC 5737 TEST-NET-1) must time out gracefully within ~2.5s and not hang
	start := time.Now()
	// 192.0.2.1 is reserved for documentation/testing and drops packets
	client := redis.NewClient("192.0.2.1:6379")
	duration := time.Since(start)

	if client == nil {
		t.Fatal("expected non-nil redis.Client on timeout")
	}
	defer client.Close()

	t.Logf("NewClient with blackhole address returned in %v", duration)
	if duration > 5*time.Second {
		t.Errorf("FAIL: NewClient took %v, exceeded maximum timeout threshold of 5s", duration)
	}
}

func TestRedis_Empirical_MalformedURL(t *testing.T) {
	// 3. Malformed Redis URLs must log warning and fall back to raw address without panic
	malformedCases := []string{
		"redis://invalid user:pass@localhost:6379",
		"redis://:@@@invalid-host-format:6379",
		"rediss://[invalid-ipv6]:6379",
	}

	for _, tc := range malformedCases {
		client := redis.NewClient(tc)
		if client == nil {
			t.Errorf("expected non-nil client for malformed URL %q", tc)
		} else {
			client.Close()
		}
	}
}

func TestRedis_Empirical_EnvVarFallbackHierarchy(t *testing.T) {
	// 4. Verify env var fallback order
	origURL := os.Getenv("REDIS_URL")
	origAddr := os.Getenv("REDIS_ADDR")
	defer func() {
		os.Setenv("REDIS_URL", origURL)
		os.Setenv("REDIS_ADDR", origAddr)
	}()

	// Case A: Both empty -> localhost:6379
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("REDIS_ADDR")
	cA := redis.NewClient()
	if cA.Options().Addr != "localhost:6379" {
		t.Errorf("expected localhost:6379, got %s", cA.Options().Addr)
	}
	cA.Close()

	// Case B: REDIS_ADDR only -> uses REDIS_ADDR
	os.Setenv("REDIS_ADDR", "127.0.0.1:6389")
	cB := redis.NewClient()
	if cB.Options().Addr != "127.0.0.1:6389" {
		t.Errorf("expected 127.0.0.1:6389, got %s", cB.Options().Addr)
	}
	cB.Close()

	// Case C: REDIS_URL takes precedence
	os.Setenv("REDIS_URL", "redis://cluster-redis:6390/2")
	cC := redis.NewClient()
	if cC.Options().Addr != "cluster-redis:6390" {
		t.Errorf("expected cluster-redis:6390, got %s", cC.Options().Addr)
	}
	if cC.Options().DB != 2 {
		t.Errorf("expected DB 2, got %d", cC.Options().DB)
	}
	cC.Close()
}
