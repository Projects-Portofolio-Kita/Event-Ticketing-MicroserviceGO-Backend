package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/lock"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/pkg/database"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/pkg/redis"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/worker"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize database connection
	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run auto migrations for reservation schema
	if err := db.AutoMigrate(&model.Reservation{}, &model.ReservationItem{}); err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	}

	// Initialize Redis & Lock Manager
	redisClient := redis.NewClient()
	locker := lock.NewDistributedLockManager(redisClient)

	// Initialize Repository & Service
	repo := repository.NewBookingRepository(db)
	bookingService := service.NewBookingService(repo, locker, cfg.LockTTL)

	// Start Background Sweeper
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sweeper := worker.NewSweeper(bookingService, cfg.SweeperInterval)
	sweeper.Start(ctx)

	// Setup Handlers & Router
	bookingHandler := handler.NewBookingHandler(bookingService)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "booking-service",
		})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/reservations", bookingHandler.CreateReservation)
		api.GET("/reservations/:id", bookingHandler.GetReservation)
		api.POST("/reservations/:id/confirm", bookingHandler.ConfirmReservation)
		api.POST("/reservations/:id/cancel", bookingHandler.CancelReservation)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Start server in goroutine for graceful shutdown
	go func() {
		log.Printf("Booking Service listening on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server listen error: %v", err)
		}
	}()

	// Listen for termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Booking Service gracefully...")
	cancel() // stops background sweeper

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Booking Service exited cleanly.")
}
