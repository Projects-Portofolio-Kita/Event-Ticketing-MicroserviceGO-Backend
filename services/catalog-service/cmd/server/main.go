package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/seed"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// The seat availability feed it reads from booking-service is
	// service-to-service only, so catalog cannot start without the token.
	serviceToken := authz.RequireEnv("SERVICE_TOKEN")
	// Posting events requires verifying user access tokens.
	jwtSecret := authz.RequireEnv("JWT_SECRET")

	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Event{}, &model.SeatCategory{}, &model.Seat{}); err != nil {
		log.Fatal(err)
	}

	if err := seed.Run(db); err != nil {
		log.Fatal(err)
	}

	catalogRepository := repository.NewCatalogRepository(db)
	bookingClient := client.NewHTTPBookingClient()
	catalogService := service.NewCatalogService(catalogRepository, bookingClient)
	catalogHandler := handler.NewCatalogHandler(catalogService)

	router := gin.Default()
	router.Use(cors.New())

	health.Register(router, "catalog-service")

	// Unknown routes answer with the standard JSON envelope instead of gin's
	// plain-text 404.
	router.NoRoute(func(c *gin.Context) {
		httperr.JSON(c, http.StatusNotFound, "route tidak ditemukan")
	})

	api := router.Group("/api/v1")
	{
		api.GET("/events", catalogHandler.ListEvents)
		// Any signed-in user may post an event (single-account model, no roles).
		api.POST("/events", authz.RequireUser(jwtSecret), catalogHandler.CreateEvent)
		// Static path before /:id — gin matches the literal route first.
		api.GET("/events/cities", catalogHandler.ListCities)
		api.GET("/events/:id", catalogHandler.GetEvent)
		api.GET("/events/:id/seatmap", catalogHandler.GetSeatMap)
		// Public tier availability for TIER events.
		api.GET("/events/:id/ticket-types", catalogHandler.GetTicketTypes)
		// Internal stock feed read by booking-service when it validates a
		// tier reservation (X-Service-Token, never reachable from the gateway).
		api.GET("/events/:id/tier-quotas", authz.RequireService(serviceToken), catalogHandler.GetTierQuotas)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8082"
	}

	log.Println("Catalog Service running on :" + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
