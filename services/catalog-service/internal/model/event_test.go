package model_test

import (
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
)

func TestEvent_CanTransitionTo_ValidTransitions(t *testing.T) {
	// Table-driven tests for all valid transitions
	validCases := []struct {
		name string
		from model.EventStatus
		to   model.EventStatus
	}{
		{"DraftToPublished", model.StatusDraft, model.StatusPublished},
		{"DraftToCancelled", model.StatusDraft, model.StatusCancelled},
		{"PublishedToOnSale", model.StatusPublished, model.StatusOnSale},
		{"PublishedToCancelled", model.StatusPublished, model.StatusCancelled},
		{"OnSaleToSoldOut", model.StatusOnSale, model.StatusSoldOut},
		{"OnSaleToCancelled", model.StatusOnSale, model.StatusCancelled},
		{"OnSaleToPast", model.StatusOnSale, model.StatusPast},
		{"SoldOutToOnSale", model.StatusSoldOut, model.StatusOnSale},
		{"SoldOutToCancelled", model.StatusSoldOut, model.StatusCancelled},
		{"SoldOutToPast", model.StatusSoldOut, model.StatusPast},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			e := &model.Event{Status: tc.from}
			if !e.CanTransitionTo(tc.to) {
				t.Fatalf("expected CanTransitionTo(%s) from %s to be true, got false", tc.to, tc.from)
			}

			err := e.TransitionTo(tc.to)
			if err != nil {
				t.Fatalf("expected TransitionTo(%s) from %s to succeed, got error: %v", tc.to, tc.from, err)
			}
			if e.Status != tc.to {
				t.Fatalf("expected status to be %s, got %s", tc.to, e.Status)
			}
		})
	}
}

func TestEvent_LifecycleSequence(t *testing.T) {
	event := &model.Event{
		ID:        uuid.New(),
		Name:      "Coldplay Music of the Spheres",
		Status:    model.StatusDraft,
		EventDate: time.Now().Add(30 * 24 * time.Hour),
	}

	sequence := []model.EventStatus{
		model.StatusPublished,
		model.StatusOnSale,
		model.StatusSoldOut,
		model.StatusOnSale, // Restocked / additional seats released
		model.StatusPast,   // Event completed
	}

	for _, nextStatus := range sequence {
		if !event.CanTransitionTo(nextStatus) {
			t.Fatalf("failed transition from %s to %s", event.Status, nextStatus)
		}
		if err := event.TransitionTo(nextStatus); err != nil {
			t.Fatalf("unexpected error during transition from %s to %s: %v", event.Status, nextStatus, err)
		}
	}

	if event.Status != model.StatusPast {
		t.Fatalf("expected final status to be PAST, got %s", event.Status)
	}
}

func TestEvent_CanTransitionTo_InvalidTransitions(t *testing.T) {
	invalidCases := []struct {
		name string
		from model.EventStatus
		to   model.EventStatus
	}{
		// From DRAFT
		{"DraftToOnSale", model.StatusDraft, model.StatusOnSale},
		{"DraftToSoldOut", model.StatusDraft, model.StatusSoldOut},
		{"DraftToPast", model.StatusDraft, model.StatusPast},
		{"DraftToDraft", model.StatusDraft, model.StatusDraft},

		// From PUBLISHED
		{"PublishedToDraft", model.StatusPublished, model.StatusDraft},
		{"PublishedToSoldOut", model.StatusPublished, model.StatusSoldOut},
		{"PublishedToPast", model.StatusPublished, model.StatusPast},
		{"PublishedToPublished", model.StatusPublished, model.StatusPublished},

		// From ON_SALE
		{"OnSaleToDraft", model.StatusOnSale, model.StatusDraft},
		{"OnSaleToPublished", model.StatusOnSale, model.StatusPublished},
		{"OnSaleToOnSale", model.StatusOnSale, model.StatusOnSale},

		// From SOLD_OUT
		{"SoldOutToDraft", model.StatusSoldOut, model.StatusDraft},
		{"SoldOutToPublished", model.StatusSoldOut, model.StatusPublished},
		{"SoldOutToSoldOut", model.StatusSoldOut, model.StatusSoldOut},

		// From CANCELLED (Terminal)
		{"CancelledToDraft", model.StatusCancelled, model.StatusDraft},
		{"CancelledToPublished", model.StatusCancelled, model.StatusPublished},
		{"CancelledToOnSale", model.StatusCancelled, model.StatusOnSale},
		{"CancelledToSoldOut", model.StatusCancelled, model.StatusSoldOut},
		{"CancelledToCancelled", model.StatusCancelled, model.StatusCancelled},
		{"CancelledToPast", model.StatusCancelled, model.StatusPast},

		// From PAST (Terminal)
		{"PastToDraft", model.StatusPast, model.StatusDraft},
		{"PastToPublished", model.StatusPast, model.StatusPublished},
		{"PastToOnSale", model.StatusPast, model.StatusOnSale},
		{"PastToSoldOut", model.StatusPast, model.StatusSoldOut},
		{"PastToCancelled", model.StatusPast, model.StatusCancelled},
		{"PastToPast", model.StatusPast, model.StatusPast},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			e := &model.Event{Status: tc.from}
			if e.CanTransitionTo(tc.to) {
				t.Fatalf("expected CanTransitionTo(%s) from %s to be false, got true", tc.to, tc.from)
			}

			err := e.TransitionTo(tc.to)
			if err == nil {
				t.Fatalf("expected TransitionTo(%s) from %s to fail, but got nil", tc.to, tc.from)
			}

			if tc.from == model.StatusCancelled || tc.from == model.StatusPast {
				if err != model.ErrTerminalState {
					t.Fatalf("expected ErrTerminalState for terminal status %s, got: %v", tc.from, err)
				}
			} else {
				if err != model.ErrInvalidStateTransition {
					t.Fatalf("expected ErrInvalidStateTransition, got: %v", err)
				}
			}
		})
	}
}

func TestEvent_PricingTierLockRule(t *testing.T) {
	// Modifying tiers is allowed in DRAFT and PUBLISHED only
	unlockedStatuses := []model.EventStatus{
		model.StatusDraft,
		model.StatusPublished,
	}

	for _, st := range unlockedStatuses {
		e := &model.Event{Status: st}
		if !e.CanModifyPricingTiers() {
			t.Fatalf("expected CanModifyPricingTiers to be true for status %s", st)
		}
	}

	// Modifying tiers is locked in ON_SALE, SOLD_OUT, CANCELLED, PAST
	lockedStatuses := []model.EventStatus{
		model.StatusOnSale,
		model.StatusSoldOut,
		model.StatusCancelled,
		model.StatusPast,
	}

	for _, st := range lockedStatuses {
		e := &model.Event{Status: st}
		if e.CanModifyPricingTiers() {
			t.Fatalf("expected CanModifyPricingTiers to be false for status %s", st)
		}
	}
}

func TestPricingTier_Validation(t *testing.T) {
	t.Run("Valid Tier", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "VIP",
			Price:      1500000,
			Currency:   "IDR",
			TotalSeats: 100,
		}
		if err := pt.Validate(); err != nil {
			t.Fatalf("expected valid tier to pass, got: %v", err)
		}
	})

	t.Run("Valid Free Tier (Price == 0)", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "General Admission Free",
			Price:      0,
			Currency:   "IDR",
			TotalSeats: 50,
		}
		if err := pt.Validate(); err != nil {
			t.Fatalf("expected free tier (price 0) to pass, got: %v", err)
		}
	})

	t.Run("Default Currency Fallback", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "Regular",
			Price:      500000,
			Currency:   "",
			TotalSeats: 200,
		}
		if err := pt.Validate(); err != nil {
			t.Fatalf("expected validation to succeed, got: %v", err)
		}
		if pt.Currency != "IDR" {
			t.Fatalf("expected Currency to default to 'IDR', got: %s", pt.Currency)
		}
	})

	t.Run("Negative Price Rejected", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "Invalid Tier",
			Price:      -100,
			Currency:   "IDR",
			TotalSeats: 100,
		}
		if err := pt.Validate(); err != model.ErrInvalidTierPrice {
			t.Fatalf("expected ErrInvalidTierPrice, got: %v", err)
		}
	})

	t.Run("Zero Seats Rejected", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "Zero Seats",
			Price:      100000,
			Currency:   "IDR",
			TotalSeats: 0,
		}
		if err := pt.Validate(); err != model.ErrInvalidTierSeats {
			t.Fatalf("expected ErrInvalidTierSeats, got: %v", err)
		}
	})

	t.Run("Negative Seats Rejected", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "Negative Seats",
			Price:      100000,
			Currency:   "IDR",
			TotalSeats: -10,
		}
		if err := pt.Validate(); err != model.ErrInvalidTierSeats {
			t.Fatalf("expected ErrInvalidTierSeats, got: %v", err)
		}
	})

	t.Run("Empty Tier Name Rejected", func(t *testing.T) {
		pt := &model.PricingTier{
			TierName:   "",
			Price:      100000,
			Currency:   "IDR",
			TotalSeats: 50,
		}
		if err := pt.Validate(); err != model.ErrInvalidTierName {
			t.Fatalf("expected ErrInvalidTierName, got: %v", err)
		}
	})

	t.Run("Whitespace Only Tier Name Rejected", func(t *testing.T) {
		whitespaceNames := []struct {
			name  string
			value string
		}{
			{"Single space", " "},
			{"Multiple spaces", "   "},
			{"Tab character", "\t"},
			{"Newline character", "\n"},
			{"Mixed whitespace", " \t \r\n  "},
		}

		for _, tc := range whitespaceNames {
			t.Run(tc.name, func(t *testing.T) {
				pt := &model.PricingTier{
					TierName:   tc.value,
					Price:      100000,
					Currency:   "IDR",
					TotalSeats: 50,
				}
				err := pt.Validate()
				if err != model.ErrInvalidTierName {
					t.Fatalf("expected ErrInvalidTierName for %q, got: %v", tc.value, err)
				}
			})
		}
	})
}

func TestVenue_Model(t *testing.T) {
	venue := &model.Venue{
		ID:            uuid.New(),
		Name:          "Gelora Bung Karno",
		Address:       "Senayan",
		City:          "Jakarta Pusat",
		TotalCapacity: 77000,
		LayoutConfig:  `{"type":"stadium"}`,
	}

	if venue.TableName() != "venues" {
		t.Fatalf("expected table name 'venues', got %s", venue.TableName())
	}
	if venue.TotalCapacity != 77000 {
		t.Fatalf("expected capacity 77000, got %d", venue.TotalCapacity)
	}
}

func TestEvent_NilReceiverSafety(t *testing.T) {
	var e *model.Event

	t.Run("CanTransitionTo on nil Event returns false without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil Event.CanTransitionTo: %v", r)
			}
		}()
		if e.CanTransitionTo(model.StatusPublished) {
			t.Errorf("expected CanTransitionTo on nil Event to return false, got true")
		}
	})

	t.Run("TransitionTo on nil Event returns ErrNilEvent without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil Event.TransitionTo: %v", r)
			}
		}()
		err := e.TransitionTo(model.StatusPublished)
		if err != model.ErrNilEvent {
			t.Errorf("expected ErrNilEvent on nil Event.TransitionTo, got: %v", err)
		}
	})

	t.Run("IsTerminal on nil Event returns false without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil Event.IsTerminal: %v", r)
			}
		}()
		if e.IsTerminal() {
			t.Errorf("expected IsTerminal on nil Event to return false, got true")
		}
	})

	t.Run("CanModifyPricingTiers on nil Event returns false without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil Event.CanModifyPricingTiers: %v", r)
			}
		}()
		if e.CanModifyPricingTiers() {
			t.Errorf("expected CanModifyPricingTiers on nil Event to return false, got true")
		}
	})

	t.Run("BeforeCreate on nil Event returns ErrNilEvent without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil Event.BeforeCreate: %v", r)
			}
		}()
		err := e.BeforeCreate(nil)
		if err != model.ErrNilEvent {
			t.Errorf("expected ErrNilEvent on nil Event.BeforeCreate, got: %v", err)
		}
	})
}

func TestPricingTier_NilReceiverSafety(t *testing.T) {
	var pt *model.PricingTier

	t.Run("Validate on nil PricingTier returns ErrNilPricingTier without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil PricingTier.Validate: %v", r)
			}
		}()
		err := pt.Validate()
		if err != model.ErrNilPricingTier {
			t.Errorf("expected ErrNilPricingTier on nil PricingTier.Validate, got: %v", err)
		}
	})

	t.Run("BeforeCreate on nil PricingTier returns ErrNilPricingTier without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic on nil PricingTier.BeforeCreate: %v", r)
			}
		}()
		err := pt.BeforeCreate(nil)
		if err != model.ErrNilPricingTier {
			t.Errorf("expected ErrNilPricingTier on nil PricingTier.BeforeCreate, got: %v", err)
		}
	})
}
