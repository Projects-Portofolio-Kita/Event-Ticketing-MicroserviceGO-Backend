// Package jwt provides JWT authentication claims and validation utilities.
//
// File: jwt.go
// Usability: Declares JWTClaims structure and ValidateToken helper for verifying cryptographic HMAC signatures.
package jwt

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims encapsulates custom user identity claims embedded in access/refresh tokens.
type JWTClaims struct {
	UserID uuid.UUID      `json:"user_id"`
	Email  string         `json:"email"`
	Role   model.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// ValidateToken parses, cryptographically validates signature, and extracts JWTClaims.
func ValidateToken(tokenString string, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return claims, nil
}
