package dto

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// bindJSON runs the exact binding path the handlers use (ShouldBindJSON).
func bindJSON(t *testing.T, target any, body string) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c.ShouldBindJSON(target)
}

// TestRegisterRequestBinding covers the validation the service itself does not
// perform, e.g. the minimum password length.
func TestRegisterRequestBinding(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid request", `{"name":"Alice","email":"alice@example.com","password":"secret1"}`, false},
		{"valid request with canPostEvent", `{"name":"Alice","email":"alice@example.com","password":"secret1","canPostEvent":true}`, false},
		{"password below minimum length", `{"name":"Alice","email":"alice@example.com","password":"abc"}`, true},
		{"missing password", `{"name":"Alice","email":"alice@example.com"}`, true},
		{"invalid email", `{"name":"Alice","email":"not-an-email","password":"secret1"}`, true},
		{"name too short", `{"name":"A","email":"alice@example.com","password":"secret1"}`, true},
		{"empty body", `{}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bindJSON(t, &RegisterRequest{}, tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("bind error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginRequestBinding(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid request", `{"email":"alice@example.com","password":"secret1"}`, false},
		{"missing password", `{"email":"alice@example.com"}`, true},
		{"invalid email", `{"email":"alice","password":"secret1"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bindJSON(t, &LoginRequest{}, tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("bind error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestRefreshRequestBinding(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid request", `{"refreshToken":"0123456789abcdef"}`, false},
		{"token too short", `{"refreshToken":"abc"}`, true},
		{"missing token", `{}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bindJSON(t, &RefreshRequest{}, tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("bind error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
