package main

import (
	"log"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/middleware"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Configurations
	cfg := config.LoadConfig()

	// 2. Initialize Database Connection
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Printf("[Warning] Failed to connect to database (%v). Ensure postgres container is running.\n", err)
	} else {
		log.Println("[Database] Successfully connected to PostgreSQL")
	}

	// 3. Initialize Repositories
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)

	// 4. Initialize Services
	authService := service.NewAuthService(userRepo, tokenRepo, cfg)
	userService := service.NewUserService(userRepo, tokenRepo)

	// 5. Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	// 6. Setup Gin Router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		if db == nil {
			dbStatus = "disconnected"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"service":  "user-service",
			"database": dbStatus,
		})
	})

	// API v1 Routes
	v1 := router.Group("/api/v1")
	{
		// Public Auth Endpoints
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
			authGroup.POST("/forgot-password", authHandler.ForgotPassword)
			authGroup.POST("/reset-password", authHandler.ResetPassword)

			// Protected Logout
			authGroup.POST("/logout", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.Logout)
		}

		// User Profile Endpoints (Protected by JWT)
		usersGroup := v1.Group("/users")
		usersGroup.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			usersGroup.GET("/me", userHandler.GetProfile)
			usersGroup.PATCH("/me", userHandler.UpdateProfile)
			usersGroup.DELETE("/me", userHandler.DeactivateAccount)

			// Admin only endpoint
			usersGroup.GET("/:id", middleware.RequireRoles(model.RoleAdmin), userHandler.GetUserByID)
		}
	}

	port := ":" + cfg.ServerPort
	log.Printf("[Server] user-service starting on %s\n", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
