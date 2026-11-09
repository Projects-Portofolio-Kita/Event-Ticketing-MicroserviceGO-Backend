// Package main provides the application entry point and server wiring for payment-service.
//
// File: main.go
// Usability: Bootstraps dependencies, runs database auto-migrations, and initializes services.
package main

import (
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/pkg/database"
)

func main() {
	db, err := database.NewConnection()
	if err != nil {
		log.Printf("[Warning] Failed to connect to database (%v)\n", err)
	} else if err := db.AutoMigrate(&model.Payment{}); err != nil {
		log.Printf("[Warning] Auto-migration failed: %v\n", err)
	}

	repo := repository.NewPaymentRepository(db)
	qris := provider.NewMockQRISProvider()
	booking := service.NewHTTPBookingClient("")
	notif := service.NewHTTPNotificationClient("")
	svc := service.NewPaymentService(repo, qris, booking, notif)
	h := handler.NewPaymentHandler(svc)

	router := setupRouter(h)
	startServer(router)
}
