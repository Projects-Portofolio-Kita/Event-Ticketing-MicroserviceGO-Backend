package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test_secret_for_middleware_12345678"

	user := &model.User{
		ID:    uuid.New(),
		Email: "user@test.com",
		Role:  model.RoleCustomer,
	}
	validToken, _, _ := jwt.GenerateAccessToken(user, secret, 15)

	r := gin.New()
	r.Use(AuthMiddleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// 1. Missing Authorization Header
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}

	// 2. Valid Authorization Header
	req, _ = http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestRequireRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		roleHeader := c.GetHeader("X-Role")
		if roleHeader != "" {
			c.Set("role", model.UserRole(roleHeader))
		}
		c.Next()
	})

	r.GET("/admin-only", RequireRoles(model.RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 1. Customer tries to access admin-only
	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("X-Role", "CUSTOMER")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}

	// 2. Admin accesses admin-only
	req, _ = http.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("X-Role", "ADMIN")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
