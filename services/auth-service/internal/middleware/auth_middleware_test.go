package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authjwt "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/jwt"
	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret"

func init() { gin.SetMode(gin.TestMode) }

// newRouter wires RequireAuth in front of a handler that echoes the user ID.
func newRouter(manager *authjwt.Manager) *gin.Engine {
	r := gin.New()
	r.GET("/me", RequireAuth(manager), func(c *gin.Context) {
		// RequireAuth stores a uuid.UUID, not a string.
		raw, _ := c.Get("userID")
		id, _ := raw.(uuid.UUID)
		c.JSON(http.StatusOK, gin.H{"userID": id.String()})
	})
	return r
}

func doRequest(r *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return body["message"]
}

// signClaims forges an HS256 token with the test secret.
func signClaims(t *testing.T, claims jwtlib.MapClaims) string {
	t.Helper()
	raw, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return raw
}

func TestRequireAuthAcceptsValidToken(t *testing.T) {
	manager := authjwt.NewManager(testSecret, 15*time.Minute, time.Hour)
	userID := uuid.New()
	token, err := manager.GenerateAccess(userID, "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}

	w := doRequest(newRouter(manager), token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["userID"] != userID.String() {
		t.Errorf("userID = %q, want %q", body["userID"], userID.String())
	}
}

func TestRequireAuthRejectsBadRequests(t *testing.T) {
	manager := authjwt.NewManager(testSecret, 15*time.Minute, time.Hour)
	valid, err := manager.GenerateAccess(uuid.New(), "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}

	other := authjwt.NewManager("another-secret", 15*time.Minute, time.Hour)
	foreign, err := other.GenerateAccess(uuid.New(), "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}

	now := time.Now()
	nonUUID := signClaims(t, jwtlib.MapClaims{
		"sub":   "not-a-uuid",
		"email": "alice@example.com",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	})

	tests := []struct {
		name  string
		token string
	}{
		{"missing header", ""},
		{"garbage token", "not.a.jwt"},
		{"signed with another secret", foreign},
		{"valid token with non-UUID subject", nonUUID},
		{"tampered token", valid + "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(newRouter(manager), tt.token)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body = %s", w.Code, w.Body.String())
			}
			if decodeMessage(t, w) == "" {
				t.Error("error envelope has no message")
			}
		})
	}
}
