package jwt

import (
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/google/uuid"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	secret := "my_test_jwt_secret_key_123456789012"
	user := &model.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  model.RoleCustomer,
	}

	tokenStr, expiresIn, err := GenerateAccessToken(user, secret, 15)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresIn != int64(15*60) {
		t.Fatalf("expected 900 seconds, got %d", expiresIn)
	}

	// Validate valid token
	claims, err := ValidateToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected user ID %v, got %v", user.ID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Fatalf("expected email %v, got %v", user.Email, claims.Email)
	}
	if claims.Role != user.Role {
		t.Fatalf("expected role %v, got %v", user.Role, claims.Role)
	}

	// Validate with wrong secret
	_, err = ValidateToken(tokenStr, "wrong_secret_key")
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	secret := "my_test_jwt_secret_key_123456789012"
	user := &model.User{
		ID:    uuid.New(),
		Email: "refresh@example.com",
		Role:  model.RoleAdmin,
	}

	tokenStr, expiresAt, err := GenerateRefreshToken(user, secret, 7)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expected expiration in the future")
	}

	claims, err := ValidateToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected user ID %v, got %v", user.ID, claims.UserID)
	}
}
