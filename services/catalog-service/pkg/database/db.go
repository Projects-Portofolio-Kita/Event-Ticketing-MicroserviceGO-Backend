// Package database provides PostgreSQL connection management and schema migration routines.
//
// File: db.go
// Usability: Connects to PostgreSQL, configures connection pool, and executes schema auto-migration.
package database

import (
	"fmt"
	"os"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" { databaseURL = os.Getenv("DATABASE_URL") }
	if databaseURL == "" { databaseURL = "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable" }

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil { return nil, fmt.Errorf("failed to connect to database: %w", err) }

	sqlDB, err := db.DB()
	if err != nil { return nil, fmt.Errorf("failed to retrieve sql.DB: %w", err) }

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

func NewConnection() (*gorm.DB, error) { return Connect("") }

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.Venue{}, &model.Event{}, &model.PricingTier{}); err != nil {
		return fmt.Errorf("failed to auto-migrate database schema: %w", err)
	}
	return nil
}