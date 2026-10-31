// Package main provides the application entry point and server wiring for catalog-service.
//
// File: main.go
// Usability: Bootstraps dependencies, connects DB and Redis, registers routes, and invokes the HTTP server.
package main

import (
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	pkgredis "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	db := initDatabase(cfg.DatabaseURL)
	redisClient := pkgredis.NewClient(cfg.RedisURL)
	cacheService := service.NewRedisCacheService(redisClient, 60*time.Second)

	var eventRepo repository.EventRepository
	var venueRepo repository.VenueRepository
	var tierRepo repository.PricingTierRepository
	if db != nil {
		eventRepo = repository.NewEventRepository(db)
		venueRepo = repository.NewVenueRepository(db)
		tierRepo = repository.NewPricingTierRepository(db)
	}

	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	tierService := service.NewPricingTierService(tierRepo, eventRepo, cacheService)

	healthHandler := handler.NewHealthHandler()
	eventHandler := handler.NewEventHandler(eventService)
	tierHandler := handler.NewPricingTierHandler(tierService)

	router := gin.Default()
	handler.RegisterRoutes(router, eventHandler, tierHandler, healthHandler)
	startServerWithGracefulShutdown(cfg.ServerPort, router, db, redisClient)
}
