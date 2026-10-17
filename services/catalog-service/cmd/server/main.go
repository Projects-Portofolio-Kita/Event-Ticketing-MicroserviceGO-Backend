package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/database"
	pkgredis "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	// Connect PostgreSQL with graceful degradation if offline
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Printf("[WARN] Failed to connect to database (%s): %v. Running in degraded mode.", cfg.DatabaseURL, err)
	} else {
		log.Println("[INFO] Connected to PostgreSQL database")
		if err := database.AutoMigrate(db); err != nil {
			log.Printf("[WARN] AutoMigrate failed: %v", err)
		} else {
			log.Println("[INFO] Database schema auto-migrated successfully")
		}
		if err := database.SeedVenues(db); err != nil {
			log.Printf("[WARN] SeedVenues failed: %v", err)
		}
	}

	// Connect Redis with fallback resilience
	redisClient := pkgredis.NewClient(cfg.RedisURL)
	cacheService := service.NewRedisCacheService(redisClient, 60*time.Second)

	// Wire Repositories
	var eventRepo repository.EventRepository
	var venueRepo repository.VenueRepository
	var tierRepo repository.PricingTierRepository

	if db != nil {
		eventRepo = repository.NewEventRepository(db)
		venueRepo = repository.NewVenueRepository(db)
		tierRepo = repository.NewPricingTierRepository(db)
	}

	// Wire Services
	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	tierService := service.NewPricingTierService(tierRepo, eventRepo, cacheService)

	// Wire Handlers
	healthHandler := handler.NewHealthHandler()
	eventHandler := handler.NewEventHandler(eventService)
	tierHandler := handler.NewPricingTierHandler(tierService)

	// Gin Router
	router := gin.Default()
	handler.RegisterRoutes(router, eventHandler, tierHandler, healthHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Printf("[INFO] Catalog Service running on port :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] ListenAndServe error: %v", err)
		}
	}()

	// Graceful shutdown handling SIGINT and SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[INFO] Shutting down catalog service server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[WARN] Server forced to shutdown: %v", err)
	}

	if redisClient != nil {
		_ = redisClient.Close()
	}

	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}

	log.Println("[INFO] Catalog Service exited cleanly")
}
