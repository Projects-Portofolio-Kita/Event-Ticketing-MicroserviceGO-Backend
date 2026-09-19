package lock

import (
	"context"
	"testing"
	"time"
)

func TestDistributedLockManager_FallbackLockAndRelease(t *testing.T) {
	// Initialize without redis client to test in-memory fallback
	mgr := NewDistributedLockManager(nil)
	ctx := context.Background()

	eventID := "event-1"
	seatID := "seat-A1"
	holder1 := "user-1"
	holder2 := "user-2"
	ttl := 1 * time.Second

	// 1. First lock acquisition should succeed
	locked, err := mgr.LockSeat(ctx, eventID, seatID, holder1, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Fatalf("expected lock to be acquired")
	}

	// 2. Second lock acquisition while first is held should fail
	locked2, err := mgr.LockSeat(ctx, eventID, seatID, holder2, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked2 {
		t.Fatalf("expected lock acquisition to fail for another user")
	}

	// 3. Release lock
	err = mgr.ReleaseSeat(ctx, eventID, seatID)
	if err != nil {
		t.Fatalf("failed to release seat: %v", err)
	}

	// 4. Now holder2 should be able to acquire lock
	locked3, err := mgr.LockSeat(ctx, eventID, seatID, holder2, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked3 {
		t.Fatalf("expected lock to be acquired after release")
	}
}

func TestDistributedLockManager_BatchRelease(t *testing.T) {
	mgr := NewDistributedLockManager(nil)
	ctx := context.Background()
	eventID := "event-batch"
	seats := []string{"s1", "s2", "s3"}

	for _, s := range seats {
		locked, err := mgr.LockSeat(ctx, eventID, s, "user-batch", 10*time.Second)
		if err != nil || !locked {
			t.Fatalf("failed to lock seat %s", s)
		}
	}

	if err := mgr.ReleaseSeats(ctx, eventID, seats); err != nil {
		t.Fatalf("failed to release seats: %v", err)
	}

	// Verify all can be acquired again
	for _, s := range seats {
		locked, err := mgr.LockSeat(ctx, eventID, s, "user-new", 10*time.Second)
		if err != nil || !locked {
			t.Fatalf("expected seat %s to be available after batch release", s)
		}
	}
}
