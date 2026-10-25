package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// Fail fast: notifications carry user data, so the service must never come
	// up without its credentials.
	jwtSecret := authz.RequireEnv("JWT_SECRET")
	serviceToken := authz.RequireEnv("SERVICE_TOKEN")

	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		log.Fatal(err)
	}

	notificationRepository := repository.NewNotificationRepository(db)
	notificationService := service.NewNotificationService(notificationRepository)
	notificationHandler := handler.NewNotificationHandler(notificationService)

	router := gin.Default()
	router.Use(cors.New())

	health.Register(router, "notification-service")

	// Unknown routes answer with the standard JSON envelope instead of gin's
	// plain-text 404.
	router.NoRoute(func(c *gin.Context) {
		httperr.JSON(c, http.StatusNotFound, "route tidak ditemukan")
	})

	api := router.Group("/api/v1")
	{
		// Writes come from booking/payment via X-Service-Token; reads and
		// mark-read belong to the signed-in user (ownership enforced by the
		// service against the token's subject).
		api.POST("/notifications", authz.RequireService(serviceToken), notificationHandler.Create)
		api.GET("/notifications", authz.RequireUser(jwtSecret), notificationHandler.List)
		api.POST("/notifications/:id/read", authz.RequireUser(jwtSecret), notificationHandler.MarkRead)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8085"
	}

	log.Println("Notification Service running on :" + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
