package jwt

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret"

func newTestManager(accessTTL, refreshTTL time.Duration) *Manager {
	return NewManager(testSecret, accessTTL, refreshTTL)
}

// defaultClaims returns a fresh, non-expired claim set for forged tokens.
func defaultClaims(sub, email string) jwtlib.MapClaims {
	now := time.Now()
	return jwtlib.MapClaims{
		"sub":   sub,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}
}

func sign(t *testing.T, method jwtlib.SigningMethod, secret string, claims jwtlib.MapClaims) string {
	t.Helper()
	raw, err := jwtlib.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return raw
}

func b64(t *testing.T, payload string) string {
	t.Helper()
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func TestAccessAndRefreshTTLPassthrough(t *testing.T) {
	m := newTestManager(15*time.Minute, 7*24*time.Hour)
	if m.AccessTTL() != 15*time.Minute {
		t.Errorf("AccessTTL() = %v, want 15m", m.AccessTTL())
	}
	if m.RefreshTTL() != 7*24*time.Hour {
		t.Errorf("RefreshTTL() = %v, want 168h", m.RefreshTTL())
	}
}

func TestGenerateAccessRoundTrip(t *testing.T) {
	m := newTestManager(15*time.Minute, time.Hour)
	userID := uuid.New()

	raw, err := m.GenerateAccess(userID, "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}

	sub, email, err := m.ParseAccess(raw)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if sub != userID.String() {
		t.Errorf("sub = %q, want %q", sub, userID.String())
	}
	if email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", email)
	}
}

func TestParseAccessRejectsExpiredToken(t *testing.T) {
	t.Run("ttl elapses", func(t *testing.T) {
		m := newTestManager(5*time.Millisecond, time.Hour)
		raw, err := m.GenerateAccess(uuid.New(), "alice@example.com")
		if err != nil {
			t.Fatalf("GenerateAccess: %v", err)
		}
		time.Sleep(20 * time.Millisecond)

		if _, _, err := m.ParseAccess(raw); err == nil {
			t.Fatal("ParseAccess = nil error, want expired-token rejection")
		}
	})

	t.Run("hand-crafted exp in the past", func(t *testing.T) {
		m := newTestManager(15*time.Minute, time.Hour)
		now := time.Now()
		raw := sign(t, jwtlib.SigningMethodHS256, testSecret, jwtlib.MapClaims{
			"sub":   uuid.New().String(),
			"email": "alice@example.com",
			"iat":   now.Add(-time.Hour).Unix(),
			"exp":   now.Add(-time.Minute).Unix(),
		})

		if _, _, err := m.ParseAccess(raw); err == nil {
			t.Fatal("ParseAccess = nil error, want expired-token rejection")
		}
	})
}

func TestParseAccessRejectsForeignSecret(t *testing.T) {
	other := NewManager("a-completely-different-secret", 15*time.Minute, time.Hour)
	raw, err := other.GenerateAccess(uuid.New(), "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}

	m := newTestManager(15*time.Minute, time.Hour)
	if _, _, err := m.ParseAccess(raw); err == nil {
		t.Fatal("ParseAccess = nil error, want signature rejection for a foreign secret")
	}
}

func TestParseAccessRejectsTamperedTokens(t *testing.T) {
	m := newTestManager(15*time.Minute, time.Hour)
	sub := uuid.New().String()
	valid, err := m.GenerateAccess(uuid.MustParse(sub), "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}
	parts := strings.Split(valid, ".")

	tests := []struct {
		name  string
		token string
	}{
		{"alg none without signature", b64(t, `{"alg":"none","typ":"JWT"}`) + "." + b64(t, `{"sub":"`+sub+`","email":"alice@example.com"}`) + "."},
		{"alg none with signature segment", b64(t, `{"alg":"none","typ":"JWT"}`) + "." + b64(t, `{"sub":"`+sub+`","email":"alice@example.com"}`) + ".x"},
		{
			"payload edited after signing",
			parts[0] + "." + b64(t, `{"sub":"`+sub+`","email":"attacker@evil.com","iat":1,"exp":9999999999}`) + "." + parts[2],
		},
		{
			"header swapped to HS512 with another secret",
			sign(t, jwtlib.SigningMethodHS512, "not-the-manager-secret", defaultClaims(sub, "alice@example.com")),
		},
		{"garbage string", "not.a.jwt"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := m.ParseAccess(tt.token); err == nil {
				t.Fatal("ParseAccess = nil error, want rejection")
			}
		})
	}
	// ParseAccess pins WithValidMethods(["HS256"]), so even a token re-signed
	// as HS512 with the *same* secret is refused (the table above covers the
	// forged variants; see also authz's RequireUser, which does the same).
	t.Run("hs512 with the manager secret", func(t *testing.T) {
		token := sign(t, jwtlib.SigningMethodHS512, testSecret, defaultClaims(sub, "alice@example.com"))
		if _, _, err := m.ParseAccess(token); err == nil {
			t.Fatal("ParseAccess = nil error, want rejection of a non-HS256 alg")
		}
	})
}

func TestParseAccessAcceptsNonUUIDSubject(t *testing.T) {
	// UUID formatting is enforced one layer up (middleware.RequireAuth), so
	// jwt only hands the raw subject back. See auth_middleware_test.go.
	m := newTestManager(15*time.Minute, time.Hour)
	raw := sign(t, jwtlib.SigningMethodHS256, testSecret, defaultClaims("not-a-uuid", "alice@example.com"))

	sub, email, err := m.ParseAccess(raw)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if sub != "not-a-uuid" {
		t.Errorf("sub = %q, want not-a-uuid", sub)
	}
	if email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", email)
	}
}

func TestParseAccessRejectsMissingSubject(t *testing.T) {
	m := newTestManager(15*time.Minute, time.Hour)
	claims := defaultClaims("x", "alice@example.com")
	delete(claims, "sub")
	raw := sign(t, jwtlib.SigningMethodHS256, testSecret, claims)

	if _, _, err := m.ParseAccess(raw); err == nil {
		t.Fatal("ParseAccess = nil error, want missing-subject rejection")
	}
}

func TestGenerateRefreshIsOpaqueAndUnique(t *testing.T) {
	m := newTestManager(15*time.Minute, 7*24*time.Hour)

	first, err := m.GenerateRefresh()
	if err != nil {
		t.Fatalf("GenerateRefresh: %v", err)
	}
	second, err := m.GenerateRefresh()
	if err != nil {
		t.Fatalf("GenerateRefresh: %v", err)
	}

	if len(first) != 64 {
		t.Errorf("token length = %d, want 64 hex chars", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Errorf("token %q is not hex: %v", first, err)
	}
	if first == second {
		t.Error("two refresh tokens are identical, want unique randomness")
	}
}

func TestHashRefreshIsStableSHA256(t *testing.T) {
	sum := sha256.Sum256([]byte("raw-token"))
	want := hex.EncodeToString(sum[:])

	if got := HashRefresh("raw-token"); got != want {
		t.Errorf("HashRefresh = %q, want %q", got, want)
	}
	if HashRefresh("raw-token") == HashRefresh("other-token") {
		t.Error("different inputs hash to the same value")
	}
}
