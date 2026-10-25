package middleware

import (
	"net/http"
	"strings"

	authjwt "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/jwt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireAuth validates the Bearer access JWT and stores the user ID.
func RequireAuth(manager *authjwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Token tidak ditemukan. Silakan masuk."})
			return
		}
		sub, _, err := manager.ParseAccess(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Sesi kedaluwarsa. Silakan masuk ulang."})
			return
		}
		userID, err := uuid.Parse(sub)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Sesi tidak valid. Silakan masuk ulang."})
			return
		}
		c.Set("userID", userID)
		c.Next()
	}
}
