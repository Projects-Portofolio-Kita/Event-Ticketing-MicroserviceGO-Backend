// Package database provides PostgreSQL persistence and initial seeding.
//
// File: seed_ids.go
// Usability: Declares well-known deterministic UUID constants for seeded venues.
package database

import "github.com/google/uuid"

// Standard Seed Venue IDs for deterministic reference across services and tests
var (
	VenueIDGBK    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	VenueIDICEBSD = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	VenueIDJIExpo = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)
