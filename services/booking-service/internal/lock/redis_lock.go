// Package lock provides distributed locking primitives for seat reservations.
//
// File: redis_lock.go
// Usability: Implements LockSeat, ReleaseSeat, and ReleaseSeats using Redis SETNX with local memory fallback.
package lock

import (
	"context"
	"fmt"
	"time"
)

func (m *DistributedLockManager) key(eventID, seatID string) string {
	return fmt.Sprintf("seat_lock:%s:%s", eventID, seatID)
}

// LockSeat atomically locks a seat for the specified TTL using Redis SET NX EX.
func (m *DistributedLockManager) LockSeat(ctx context.Context, eventID, seatID, holderID string, ttl time.Duration) (bool, error) {
	key := m.key(eventID, seatID)
	if m.client != nil {
		if locked, err := m.client.SetNX(ctx, key, holderID, ttl).Result(); err == nil {
			return locked, nil
		}
	}
	m.fallbackMu.Lock()
	defer m.fallbackMu.Unlock()
	now := time.Now()
	if exp, exists := m.fallbackMap[key]; exists && now.Before(exp) { return false, nil }
	m.fallbackMap[key] = now.Add(ttl)
	return true, nil
}

// ReleaseSeat removes the distributed lock for a specific seat.
func (m *DistributedLockManager) ReleaseSeat(ctx context.Context, eventID, seatID string) error {
	key := m.key(eventID, seatID)
	if m.client != nil { _ = m.client.Del(ctx, key).Err() }
	m.fallbackMu.Lock()
	delete(m.fallbackMap, key)
	m.fallbackMu.Unlock()
	return nil
}

// ReleaseSeats removes distributed locks for multiple seats.
func (m *DistributedLockManager) ReleaseSeats(ctx context.Context, eventID string, seatIDs []string) error {
	for _, seatID := range seatIDs { _ = m.ReleaseSeat(ctx, eventID, seatID) }
	return nil
}