// Package service implements notification business logic and background queue dispatching.
//
// File: service_helpers.go
// Usability: Utility functions for email address validation and currency number formatting.
package service

import (
	"fmt"
	"net/mail"
	"strings"
)

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	return strings.Contains(parts[1], ".")
}

func formatAmount(amount float64) string {
	if amount != float64(int64(amount)) {
		return fmt.Sprintf("%.2f", amount)
	}
	val := int64(amount)
	if val < 0 {
		return fmt.Sprintf("%d", val)
	}
	s := fmt.Sprintf("%d", val)
	if len(s) <= 3 {
		return s
	}
	var res []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			res = append(res, ',')
		}
		res = append(res, byte(c))
	}
	return string(res)
}