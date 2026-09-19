package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Payment{}); err != nil {
		log.Fatal(err)
	}

	paymentRepository := repository.NewPaymentRepository(db)

	qrisProvider := provider.NewMockQRISProvider()

	bookingClient := service.NewMockBookingClient()

	paymentService := service.NewPaymentService(
		paymentRepository,
		qrisProvider,
		bookingClient,
	)

	paymentHandler := handler.NewPaymentHandler(paymentService)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "payment-service",
		})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/payments", paymentHandler.CreatePayment)
		api.GET("/payments/:id", paymentHandler.GetPayment)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8084"
	}

	log.Println("Payment Service running on :" + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}