package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// Fail fast: booking owns user data, so it must never come up without the
	// credentials it needs to authenticate callers.
	jwtSecret := authz.RequireEnv("JWT_SECRET")
	serviceToken := authz.RequireEnv("SERVICE_TOKEN")

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

	// Unknown routes answer with the standard JSON envelope instead of gin's
	// plain-text 404.
	router.NoRoute(func(c *gin.Context) {
		httperr.JSON(c, http.StatusNotFound, "route tidak ditemukan")
	})

	requireUser := authz.RequireUser(jwtSecret)
	requireUserOrService := authz.RequireUserOrService(jwtSecret, serviceToken)

	api := router.Group("/api/v1")
	{
		// Writes always act on the caller's own account: the service derives
		// the owner from the token, ignoring userId in the body.
		api.POST("/reservations", requireUser, reservationHandler.CreateReservation)
		// Reads, confirm and cancel are open to the owner and to
		// payment-service (X-Service-Token); the service enforces ownership.
		api.GET("/reservations/:id", requireUserOrService, reservationHandler.GetReservation)
		api.POST("/reservations/:id/confirm", requireUserOrService, reservationHandler.ConfirmReservation)
		api.POST("/reservations/:id/cancel", requireUserOrService, reservationHandler.CancelReservation)
		// Read-only availability feed consumed by catalog-service.
		api.GET("/events/:eventId/seat-status", authz.RequireService(serviceToken), reservationHandler.GetEventSeatStatus)
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
