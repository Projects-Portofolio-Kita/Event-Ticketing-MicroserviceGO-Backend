package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRegisterServesOKPayload(t *testing.T) {
	tests := []struct {
		name    string
		service string
	}{
		{"compose service name", "catalog-service"},
		{"single word service", "gateway"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			Register(r, tt.service)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var payload map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode /health payload: %v", err)
			}
			if len(payload) != 2 {
				t.Errorf("payload = %v, want exactly the status and service fields", payload)
			}
			if payload["status"] != "ok" {
				t.Errorf("status = %q, want %q", payload["status"], "ok")
			}
			if payload["service"] != tt.service {
				t.Errorf("service = %q, want %q", payload["service"], tt.service)
			}
		})
	}
}

func TestHandlerWritesExactBody(t *testing.T) {
	r := gin.New()
	r.GET("/health", Handler("auth-service"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	want := `{"status":"ok","service":"auth-service"}`
	if got := strings.TrimSpace(w.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestRegisterMountsGetOnly(t *testing.T) {
	r := gin.New()
	Register(r, "gateway")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/health", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("POST /health status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
