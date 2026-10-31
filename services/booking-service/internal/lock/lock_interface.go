// Package lock provides distributed locking primitives for seat reservations.
//
// File: lock_interface.go
// Usability: Defines the Locker interface contract and DistributedLockManager struct with fallback state.
package lock

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Locker defines operations for acquiring and releasing seat reservation locks.
type Locker interface {
	LockSeat(ctx context.Context, eventID, seatID, holderID string, ttl time.Duration) (bool, error)
	ReleaseSeat(ctx context.Context, eventID, seatID string) error
	ReleaseSeats(ctx context.Context, eventID string, seatIDs []string) error
}

// DistributedLockManager implements Locker using Redis SET NX EX with an in-memory fallback.
type DistributedLockManager struct {
	client      *redis.Client
	fallbackMu  sync.Mutex
	fallbackMap map[string]time.Time
}

// NewDistributedLockManager creates a new DistributedLockManager instance.
func NewDistributedLockManager(client *redis.Client) *DistributedLockManager {
	return &DistributedLockManager{
		client:      client,
		fallbackMap: make(map[string]time.Time),
	}
}
