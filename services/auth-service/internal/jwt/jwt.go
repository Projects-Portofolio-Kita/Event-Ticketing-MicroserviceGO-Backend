package jwt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager issues short-lived access JWTs (HS256) and opaque refresh tokens.
type Manager struct {
	secret       []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	refreshBytes int
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:       []byte(secret),
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		refreshBytes: 32,
	}
}

func (m *Manager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// GenerateAccess returns a signed JWT with sub=userID and email.
func (m *Manager) GenerateAccess(userID uuid.UUID, email string) (string, error) {
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(m.accessTTL).Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseAccess validates the JWT and returns the subject (user ID) and email.
func (m *Manager) ParseAccess(raw string) (userID string, email string, err error) {
	// Only HS256 is ever issued, so anything else (HS384/HS512 re-signed by
	// somebody who knows the secret, "none", RSA-shaped headers) is refused
	// outright rather than merely failing the keyfunc check.
	token, err := jwtlib.Parse(raw, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	}, jwtlib.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", "", err
	}
	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("invalid token")
	}
	sub, _ := claims["sub"].(string)
	mail, _ := claims["email"].(string)
	if sub == "" {
		return "", "", fmt.Errorf("token missing subject")
	}
	return sub, mail, nil
}

// GenerateRefresh returns a random opaque token (hex) for storage as a hash.
func (m *Manager) GenerateRefresh() (string, error) {
	b := make([]byte, m.refreshBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashRefresh hashes an opaque refresh token for DB lookup (SHA-256 hex).
func HashRefresh(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
