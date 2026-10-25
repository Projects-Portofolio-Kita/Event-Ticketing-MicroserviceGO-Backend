package main

import (
	"log"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
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

	api := router.Group("/api/v1")
	{
		api.GET("/events", catalogHandler.ListEvents)
		api.GET("/events/:id/seatmap", catalogHandler.GetSeatMap)
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
