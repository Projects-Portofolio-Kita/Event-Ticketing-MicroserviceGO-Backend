// Package model defines domain entities, validation rules, and GORM database mappings.
//
// File: event_fsm.go
// Usability: Implements finite state machine transition logic and capability checks (CanModifyPricingTiers, IsTerminal).
package model

// CanTransitionTo checks whether transitioning to next is valid according to the FSM matrix
func (e *Event) CanTransitionTo(next EventStatus) bool {
	if e == nil { return false }
	targets, ok := allowedTransitions[e.Status]
	if !ok { return false }
	for _, target := range targets {
		if target == next { return true }
	}
	return false
}

// TransitionTo applies the state transition if allowed, returning an appropriate domain error if not
func (e *Event) TransitionTo(next EventStatus) error {
	if e == nil { return ErrNilEvent }
	if e.IsTerminal() { return ErrTerminalState }
	if !e.CanTransitionTo(next) { return ErrInvalidStateTransition }
	e.Status = next
	return nil
}

// IsTerminal returns true if the event is in a terminal state (CANCELLED or PAST)
func (e *Event) IsTerminal() bool {
	if e == nil { return false }
	return e.Status == StatusCancelled || e.Status == StatusPast
}

// CanModifyPricingTiers returns true only if the event is in DRAFT or PUBLISHED status
func (e *Event) CanModifyPricingTiers() bool {
	if e == nil { return false }
	return e.Status == StatusDraft || e.Status == StatusPublished
}