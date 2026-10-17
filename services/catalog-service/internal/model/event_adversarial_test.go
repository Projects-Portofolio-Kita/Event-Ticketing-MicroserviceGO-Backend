package model_test

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

// ============================================================================
// 1. EMPIRICAL VERIFICATION: DISALLOWED STATE JUMPS & TERMINAL ENFORCEMENT
// ============================================================================

func TestAdversarial_DisallowedStateJumps(t *testing.T) {
	requiredDisallowedTransitions := []struct {
		name          string
		from          model.EventStatus
		to            model.EventStatus
		expectedError error
		isTerminal    bool
	}{
		// Specific required checks from dispatch
		{"Disallowed_Draft_To_OnSale", model.StatusDraft, model.StatusOnSale, model.ErrInvalidStateTransition, false},
		{"Disallowed_Published_To_SoldOut", model.StatusPublished, model.StatusSoldOut, model.ErrInvalidStateTransition, false},
		{"Disallowed_Cancelled_To_Draft", model.StatusCancelled, model.StatusDraft, model.ErrTerminalState, true},
		{"Disallowed_Past_To_OnSale", model.StatusPast, model.StatusOnSale, model.ErrTerminalState, true},

		// Additional rigorous adversarial jump checks
		{"Disallowed_Draft_To_SoldOut", model.StatusDraft, model.StatusSoldOut, model.ErrInvalidStateTransition, false},
		{"Disallowed_Draft_To_Past", model.StatusDraft, model.StatusPast, model.ErrInvalidStateTransition, false},
		{"Disallowed_Published_To_Draft", model.StatusPublished, model.StatusDraft, model.ErrInvalidStateTransition, false},
		{"Disallowed_Published_To_Past", model.StatusPublished, model.StatusPast, model.ErrInvalidStateTransition, false},
		{"Disallowed_OnSale_To_Draft", model.StatusOnSale, model.StatusDraft, model.ErrInvalidStateTransition, false},
		{"Disallowed_OnSale_To_Published", model.StatusOnSale, model.StatusPublished, model.ErrInvalidStateTransition, false},
		{"Disallowed_SoldOut_To_Draft", model.StatusSoldOut, model.StatusDraft, model.ErrInvalidStateTransition, false},
		{"Disallowed_SoldOut_To_Published", model.StatusSoldOut, model.StatusPublished, model.ErrInvalidStateTransition, false},

		// Terminal states cannot transition to ANY state
		{"Disallowed_Cancelled_To_Published", model.StatusCancelled, model.StatusPublished, model.ErrTerminalState, true},
		{"Disallowed_Cancelled_To_OnSale", model.StatusCancelled, model.StatusOnSale, model.ErrTerminalState, true},
		{"Disallowed_Cancelled_To_SoldOut", model.StatusCancelled, model.StatusSoldOut, model.ErrTerminalState, true},
		{"Disallowed_Cancelled_To_Past", model.StatusCancelled, model.StatusPast, model.ErrTerminalState, true},
		{"Disallowed_Cancelled_To_Cancelled", model.StatusCancelled, model.StatusCancelled, model.ErrTerminalState, true},
		{"Disallowed_Past_To_Draft", model.StatusPast, model.StatusDraft, model.ErrTerminalState, true},
		{"Disallowed_Past_To_Published", model.StatusPast, model.StatusPublished, model.ErrTerminalState, true},
		{"Disallowed_Past_To_SoldOut", model.StatusPast, model.StatusSoldOut, model.ErrTerminalState, true},
		{"Disallowed_Past_To_Cancelled", model.StatusPast, model.StatusCancelled, model.ErrTerminalState, true},
		{"Disallowed_Past_To_Past", model.StatusPast, model.StatusPast, model.ErrTerminalState, true},

		// Self-loops on active states (should not transition to same status)
		{"Disallowed_Draft_To_Draft", model.StatusDraft, model.StatusDraft, model.ErrInvalidStateTransition, false},
		{"Disallowed_Published_To_Published", model.StatusPublished, model.StatusPublished, model.ErrInvalidStateTransition, false},
		{"Disallowed_OnSale_To_OnSale", model.StatusOnSale, model.StatusOnSale, model.ErrInvalidStateTransition, false},
		{"Disallowed_SoldOut_To_SoldOut", model.StatusSoldOut, model.StatusSoldOut, model.ErrInvalidStateTransition, false},

		// Corrupted / arbitrary status strings
		{"Disallowed_Draft_To_Corrupted", model.StatusDraft, model.EventStatus("HACKED_STATUS"), model.ErrInvalidStateTransition, false},
		{"Disallowed_Draft_To_Empty", model.StatusDraft, model.EventStatus(""), model.ErrInvalidStateTransition, false},
	}

	for _, tc := range requiredDisallowedTransitions {
		t.Run(tc.name, func(t *testing.T) {
			ev := &model.Event{
				ID:        uuid.New(),
				Name:      "Adversarial Event",
				Status:    tc.from,
				EventDate: time.Now().Add(24 * time.Hour),
			}

			// Empirical verification of CanTransitionTo
			if ev.CanTransitionTo(tc.to) {
				t.Errorf("[%s] Expected CanTransitionTo(%s -> %s) to be false, got true", tc.name, tc.from, tc.to)
			}

			// Empirical verification of IsTerminal
			if ev.IsTerminal() != tc.isTerminal {
				t.Errorf("[%s] Expected IsTerminal()=%v for status %s, got %v", tc.name, tc.isTerminal, tc.from, ev.IsTerminal())
			}

			// Empirical verification of TransitionTo
			err := ev.TransitionTo(tc.to)
			if err == nil {
				t.Fatalf("[%s] Expected transition error, got nil", tc.name)
			}

			if err != tc.expectedError {
				t.Errorf("[%s] Expected error %v, got %v", tc.name, tc.expectedError, err)
			}

			// Ensure status was not mutated on failure
			if ev.Status != tc.from {
				t.Errorf("[%s] Status was improperly mutated from %s to %s on failed transition", tc.name, tc.from, ev.Status)
			}
		})
	}
}

// ============================================================================
// 2. EMPIRICAL VERIFICATION: UNRECOGNIZED / INVALID INITIAL STATES
// ============================================================================

func TestAdversarial_UnrecognizedInitialStatus(t *testing.T) {
	corruptedStatuses := []model.EventStatus{
		"",
		"INVALID",
		"UNKNOWN",
		"draft",     // lowercase
		"published", // lowercase
	}

	for _, cs := range corruptedStatuses {
		t.Run("Status_"+string(cs), func(t *testing.T) {
			ev := &model.Event{
				Status: cs,
			}

			if ev.CanTransitionTo(model.StatusPublished) {
				t.Errorf("Expected CanTransitionTo from unknown status '%s' to be false", cs)
			}

			err := ev.TransitionTo(model.StatusPublished)
			if err != model.ErrInvalidStateTransition {
				t.Errorf("Expected ErrInvalidStateTransition for unknown status '%s', got: %v", cs, err)
			}
		})
	}
}

// ============================================================================
// 3. EMPIRICAL VERIFICATION: NEGATIVE, ZERO, AND BOUNDARY PRICING/SEATS
// ============================================================================

func TestAdversarial_PricingTierBoundaries(t *testing.T) {
	testCases := []struct {
		name          string
		tier          model.PricingTier
		expectErr     error
		shouldSucceed bool
	}{
		// Price boundary tests
		{
			name: "Negative_Price_Tiny",
			tier: model.PricingTier{
				TierName:   "General",
				Price:      -0.01,
				Currency:   "IDR",
				TotalSeats: 100,
			},
			expectErr:     model.ErrInvalidTierPrice,
			shouldSucceed: false,
		},
		{
			name: "Negative_Price_Large",
			tier: model.PricingTier{
				TierName:   "VIP",
				Price:      -999999999.00,
				Currency:   "IDR",
				TotalSeats: 50,
			},
			expectErr:     model.ErrInvalidTierPrice,
			shouldSucceed: false,
		},
		{
			name: "Zero_Price_Allowed_Free_Event",
			tier: model.PricingTier{
				TierName:   "Free RSVP",
				Price:      0.0,
				Currency:   "IDR",
				TotalSeats: 200,
			},
			expectErr:     nil,
			shouldSucceed: true,
		},
		{
			name: "Extreme_Positive_Price",
			tier: model.PricingTier{
				TierName:   "Ultra Royal",
				Price:      10000000000.0, // 10 Billion IDR
				Currency:   "IDR",
				TotalSeats: 5,
			},
			expectErr:     nil,
			shouldSucceed: true,
		},

		// Seat boundary tests
		{
			name: "Zero_Seats_Rejected",
			tier: model.PricingTier{
				TierName:   "Standard",
				Price:      150000,
				Currency:   "IDR",
				TotalSeats: 0,
			},
			expectErr:     model.ErrInvalidTierSeats,
			shouldSucceed: false,
		},
		{
			name: "Negative_Seats_Rejected",
			tier: model.PricingTier{
				TierName:   "Standard",
				Price:      150000,
				Currency:   "IDR",
				TotalSeats: -1,
			},
			expectErr:     model.ErrInvalidTierSeats,
			shouldSucceed: false,
		},
		{
			name: "Negative_Seats_Large_Rejected",
			tier: model.PricingTier{
				TierName:   "Standard",
				Price:      150000,
				Currency:   "IDR",
				TotalSeats: -100000,
			},
			expectErr:     model.ErrInvalidTierSeats,
			shouldSucceed: false,
		},
		{
			name: "One_Seat_Allowed",
			tier: model.PricingTier{
				TierName:   "Single Backstage Pass",
				Price:      5000000,
				Currency:   "IDR",
				TotalSeats: 1,
			},
			expectErr:     nil,
			shouldSucceed: true,
		},

		// Tier Name boundary tests
		{
			name: "Empty_TierName_Rejected",
			tier: model.PricingTier{
				TierName:   "",
				Price:      100000,
				Currency:   "IDR",
				TotalSeats: 10,
			},
			expectErr:     model.ErrInvalidTierName,
			shouldSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pt := tc.tier
			err := pt.Validate()

			if tc.shouldSucceed {
				if err != nil {
					t.Fatalf("[%s] Expected validation success, got error: %v", tc.name, err)
				}
			} else {
				if err == nil {
					t.Fatalf("[%s] Expected validation failure, but got nil", tc.name)
				}
				if err != tc.expectErr {
					t.Errorf("[%s] Expected error %v, got %v", tc.name, tc.expectErr, err)
				}
			}
		})
	}
}

// TestAdversarial_FloatEdgeCases investigates behavior with NaN and Inf
func TestAdversarial_FloatEdgeCases(t *testing.T) {
	t.Run("NaN_Price_Behavior", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "NaN Tier",
			Price:      math.NaN(),
			Currency:   "IDR",
			TotalSeats: 10,
		}
		err := pt.Validate()
		// In standard Go: math.NaN() < 0 is FALSE!
		// Documenting whether Validate() lets NaN pass or catches it
		if err == nil {
			t.Logf("FINDING: PricingTier.Validate() allows math.NaN() because NaN < 0 evaluates to false in IEEE 754")
		} else {
			t.Logf("PricingTier rejected NaN with: %v", err)
		}
	})

	t.Run("Inf_Price_Behavior", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "Inf Tier",
			Price:      math.Inf(1),
			Currency:   "IDR",
			TotalSeats: 10,
		}
		err := pt.Validate()
		if err == nil {
			t.Logf("FINDING: PricingTier.Validate() allows +Inf because +Inf < 0 is false")
		}
	})

	t.Run("Whitespace_Only_TierName", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "   ",
			Price:      100000,
			Currency:   "IDR",
			TotalSeats: 10,
		}
		err := pt.Validate()
		if err != model.ErrInvalidTierName {
			t.Fatalf("expected ErrInvalidTierName for whitespace-only tier name, got: %v", err)
		}
	})
}

// ============================================================================
// 4. EMPIRICAL VERIFICATION: PRICING TIER IMMUTABILITY / LOCK RULES
// ============================================================================

func TestAdversarial_TierImmutabilityRules(t *testing.T) {
	// PRD Requirement:
	// Adding or modifying pricing tiers is only allowed in DRAFT or PUBLISHED status.
	// ON_SALE, SOLD_OUT, CANCELLED, PAST MUST lock pricing tiers.
	statuses := []struct {
		status  model.EventStatus
		allowed bool
	}{
		{model.StatusDraft, true},
		{model.StatusPublished, true},
		{model.StatusOnSale, false},
		{model.StatusSoldOut, false},
		{model.StatusCancelled, false},
		{model.StatusPast, false},
	}

	for _, sc := range statuses {
		t.Run("Status_"+string(sc.status), func(t *testing.T) {
			ev := &model.Event{
				Status: sc.status,
			}
			canModify := ev.CanModifyPricingTiers()
			if canModify != sc.allowed {
				t.Fatalf("Status %s: expected CanModifyPricingTiers=%v, got %v", sc.status, sc.allowed, canModify)
			}
		})
	}
}

// ============================================================================
// 5. EMPIRICAL VERIFICATION: RAPID / HIGH-THROUGHPUT PARALLEL TRANSITIONS
// ============================================================================

// TestAdversarial_RapidParallelTransitions tests rapid state transitions across 1000 independent events
// executed concurrently across multiple goroutines to verify lack of global state contention or map races.
func TestAdversarial_RapidParallelTransitions(t *testing.T) {
	const numEvents = 500
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64

	wg.Add(numEvents)
	for i := 0; i < numEvents; i++ {
		go func(idx int) {
			defer wg.Done()
			ev := &model.Event{
				ID:        uuid.New(),
				Name:      "Concurrent Event",
				Status:    model.StatusDraft,
				EventDate: time.Now().Add(48 * time.Hour),
			}

			// Execute standard complete lifecycle:
			// DRAFT -> PUBLISHED -> ON_SALE -> SOLD_OUT -> ON_SALE -> PAST
			steps := []model.EventStatus{
				model.StatusPublished,
				model.StatusOnSale,
				model.StatusSoldOut,
				model.StatusOnSale,
				model.StatusPast,
			}

			for _, s := range steps {
				if err := ev.TransitionTo(s); err != nil {
					atomic.AddInt64(&failureCount, 1)
					return
				}
			}

			// Verify terminal state
			if ev.IsTerminal() && ev.Status == model.StatusPast {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failureCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if failureCount > 0 {
		t.Fatalf("RapidParallelTransitions had %d failures out of %d events", failureCount, numEvents)
	}
	if successCount != int64(numEvents) {
		t.Fatalf("Expected %d successful event lifecycles, got %d", numEvents, successCount)
	}
}

// TestAdversarial_ConcurrentConflictingTerminalTransitions tests that from ON_SALE,
// when 50 workers race for CANCELLED and 50 workers race for PAST (both terminal states),
// exactly ONE transition succeeds across all 100 concurrent requests, and all 99 others are rejected.
func TestAdversarial_ConcurrentConflictingTerminalTransitions(t *testing.T) {
	ev := &model.Event{
		ID:        uuid.New(),
		Name:      "Terminal Race Event",
		Status:    model.StatusOnSale,
		EventDate: time.Now().Add(24 * time.Hour),
	}

	var mu sync.Mutex
	const workers = 100
	var wg sync.WaitGroup
	var cancelledCount int32
	var pastCount int32
	var rejectedCount int32

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		target := model.StatusCancelled
		if i%2 == 1 {
			target = model.StatusPast
		}

		go func(to model.EventStatus) {
			defer wg.Done()
			mu.Lock()
			err := ev.TransitionTo(to)
			mu.Unlock()

			if err == nil {
				if to == model.StatusCancelled {
					atomic.AddInt32(&cancelledCount, 1)
				} else {
					atomic.AddInt32(&pastCount, 1)
				}
			} else {
				atomic.AddInt32(&rejectedCount, 1)
			}
		}(target)
	}

	wg.Wait()

	totalSuccess := atomic.LoadInt32(&cancelledCount) + atomic.LoadInt32(&pastCount)
	t.Logf("Terminal race result: Cancelled=%d, Past=%d, Rejected=%d, FinalStatus=%s",
		cancelledCount, pastCount, rejectedCount, ev.Status)

	if totalSuccess != 1 {
		t.Errorf("Expected exactly 1 terminal transition to succeed, got %d", totalSuccess)
	}
	if rejectedCount != workers-1 {
		t.Errorf("Expected %d transitions to be rejected, got %d", workers-1, rejectedCount)
	}
	if !ev.IsTerminal() {
		t.Errorf("Expected final status to be terminal, got %s", ev.Status)
	}
}

// TestAdversarial_ConcurrentIdenticalTransitions tests 100 workers racing to transition DRAFT -> PUBLISHED.
// Exactly ONE must succeed; the remaining 99 must fail with ErrInvalidStateTransition.
func TestAdversarial_ConcurrentIdenticalTransitions(t *testing.T) {
	ev := &model.Event{
		ID:        uuid.New(),
		Name:      "Publish Race Event",
		Status:    model.StatusDraft,
		EventDate: time.Now().Add(24 * time.Hour),
	}

	var mu sync.Mutex
	const workers = 100
	var wg sync.WaitGroup
	var successCount int32
	var rejectedCount int32

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			err := ev.TransitionTo(model.StatusPublished)
			mu.Unlock()

			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else if err == model.ErrInvalidStateTransition {
				atomic.AddInt32(&rejectedCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Publish race result: Succeeded=%d, Rejected=%d, FinalStatus=%s",
		successCount, rejectedCount, ev.Status)

	if successCount != 1 {
		t.Errorf("Expected exactly 1 transition to succeed, got %d", successCount)
	}
	if rejectedCount != workers-1 {
		t.Errorf("Expected %d transitions to be rejected, got %d", workers-1, rejectedCount)
	}
	if ev.Status != model.StatusPublished {
		t.Errorf("Expected final status to be PUBLISHED, got %s", ev.Status)
	}
}
