package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/handler"
	authjwt "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/middleware"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func envMinutes(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return time.Duration(n) * time.Minute
	}
	return fallback
}

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set (generate one, e.g. openssl rand -hex 32)")
	}

	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.RefreshToken{}); err != nil {
		log.Fatal(err)
	}

	jwtManager := authjwt.NewManager(
		jwtSecret,
		envMinutes("JWT_ACCESS_TTL_MINUTES", 15*time.Minute),
		envMinutes("JWT_REFRESH_TTL_MINUTES", 7*24*60*time.Minute),
	)
	userRepository := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepository, jwtManager, os.Getenv("GOOGLE_CLIENT_ID"))
	authHandler := handler.NewAuthHandler(authService)

	router := gin.Default()
	router.Use(cors.New())

	health.Register(router, "auth-service")

	// Unknown routes answer with the standard JSON envelope instead of gin's
	// plain-text 404.
	router.NoRoute(func(c *gin.Context) {
		httperr.JSON(c, http.StatusNotFound, "route tidak ditemukan")
	})

	api := router.Group("/api/v1/auth")
	{
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)
		api.POST("/refresh", authHandler.Refresh)
		api.POST("/google", authHandler.Google)
		api.GET("/me", middleware.RequireAuth(jwtManager), authHandler.Me)
		// Edit the signed-in user's own profile (name / avatar URL).
		api.PATCH("/me", middleware.RequireAuth(jwtManager), authHandler.UpdateMe)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}

	log.Println("Auth Service running on :" + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
