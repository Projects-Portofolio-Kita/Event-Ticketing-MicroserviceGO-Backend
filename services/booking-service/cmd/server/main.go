package main

import (
	"log"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Reservation{}, &model.ReservationSeat{}); err != nil {
		log.Fatal(err)
	}

	reservationRepository := repository.NewReservationRepository(db)
	reservationService := service.NewReservationService(reservationRepository, notify.New())
	reservationHandler := handler.NewReservationHandler(reservationService)

	router := gin.Default()
	router.Use(cors.New())

	health.Register(router, "booking-service")

	api := router.Group("/api/v1")
	{
		api.POST("/reservations", reservationHandler.CreateReservation)
		api.GET("/reservations/:id", reservationHandler.GetReservation)
		api.POST("/reservations/:id/confirm", reservationHandler.ConfirmReservation)
		api.POST("/reservations/:id/cancel", reservationHandler.CancelReservation)
		api.GET("/events/:eventId/seat-status", reservationHandler.GetEventSeatStatus)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8083"
	}

	log.Println("Booking Service running on :" + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
