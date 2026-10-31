// Package main provides the application entry point and server wiring for booking-service.
//
// File: main.go
// Usability: Initializes config, database, redis locking, and background sweeper worker.
package main

import (
	"context"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/lock"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/pkg/database"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/pkg/redis"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/worker"
)

func main() {
	cfg := config.LoadConfig()
	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	if err := db.AutoMigrate(&model.Reservation{}, &model.ReservationItem{}); err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	}
	redisClient := redis.NewClient()
	locker := lock.NewDistributedLockManager(redisClient)
	repo := repository.NewBookingRepository(db)
	bookingService := service.NewBookingService(repo, locker, cfg.LockTTL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sweeper := worker.NewSweeper(bookingService, cfg.SweeperInterval)
	sweeper.Start(ctx)
	bookingHandler := handler.NewBookingHandler(bookingService)
	router := setupRouter(bookingHandler)
	runServerWithGracefulShutdown(cfg.ServerPort, router, cancel)
}
