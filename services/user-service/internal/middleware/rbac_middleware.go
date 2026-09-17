package middleware

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/gin-gonic/gin"
)

func RequireRoles(allowedRoles ...model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse("Access denied: role not found in token", "ERR_FORBIDDEN"))
			return
		}

		userRole, ok := roleVal.(model.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse("Access denied: invalid role format", "ERR_FORBIDDEN"))
			return
		}

		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse("Access denied: insufficient permissions", "ERR_INSUFFICIENT_PERMISSIONS"))
	}
}
