package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashing(t *testing.T) {
	rawPassword := "SecurePassword123!"

	// 1. Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// 2. Validate correct password
	err = bcrypt.CompareHashAndPassword(hashed, []byte(rawPassword))
	if err != nil {
		t.Fatalf("expected password match, got %v", err)
	}

	// 3. Validate wrong password
	err = bcrypt.CompareHashAndPassword(hashed, []byte("WrongPassword123!"))
	if err == nil {
		t.Fatal("expected password mismatch error, got nil")
	}
}

func TestHashToken(t *testing.T) {
	token := "sample-refresh-token-string"
	hash1 := hashToken(token)
	hash2 := hashToken(token)

	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Fatalf("expected identical hashes, got %s and %s", hash1, hash2)
	}
}
