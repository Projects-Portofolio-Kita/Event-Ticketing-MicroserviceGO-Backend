// Package jwt provides JWT authentication claims and validation utilities.
//
// File: jwt_token.go
// Usability: Implements HMAC-SHA256 token generation for access and refresh tokens.
package jwt

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// GenerateAccessToken mints a signed HS256 JWT access token for user authentication.
func GenerateAccessToken(user *model.User, secret string, expMinutes int) (string, int64, error) {
	expDuration := time.Duration(expMinutes) * time.Minute
	expiresAt := time.Now().Add(expDuration)
	claims := JWTClaims{
		UserID: user.ID, Email: user.Email, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "event-ticketing-user-service",
		},
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tokenString, int64(expDuration.Seconds()), err
}

// GenerateRefreshToken mints a signed HS256 refresh token for session renewal.
func GenerateRefreshToken(user *model.User, secret string, expDays int) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(expDays) * 24 * time.Hour)
	claims := JWTClaims{
		UserID: user.ID, Email: user.Email, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "event-ticketing-user-service-refresh",
		},
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tokenString, expiresAt, err
}
