// Package main provides the application entry point and server wiring for user-service.
//
// File: main.go
// Usability: Bootstraps configuration, database connection, repositories, services, and begins HTTP serving.
package main

import (
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/config"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/pkg/database"
)

func main() {
	cfg := config.LoadConfig()
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Printf("[Warning] Failed to connect to database (%v)\n", err)
	} else {
		log.Println("[Database] Successfully connected to PostgreSQL")
	}

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	authService := service.NewAuthService(userRepo, tokenRepo, cfg)
	userService := service.NewUserService(userRepo, tokenRepo)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	router := setupRouter(db, cfg, authHandler, userHandler)
	port := ":" + cfg.ServerPort
	log.Printf("[Server] user-service starting on %s\n", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
