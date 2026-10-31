// Package handler implements HTTP controllers, JSON request parsing, and status responses.
//
// File: event_handler_helpers.go
// Usability: Provides string date parsing helper across multiple standard ISO/RFC formats.
package handler

import (
	"strings"
	"time"
)

// parseQueryDate parses date strings across multiple common RFC and ISO formats
func parseQueryDate(str string) *time.Time {
	str = strings.TrimSpace(str)
	if str == "" {
		return nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, str); err == nil {
			return &t
		}
	}
	return nil
}
