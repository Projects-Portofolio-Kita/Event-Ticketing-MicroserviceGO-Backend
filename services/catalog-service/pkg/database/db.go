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

// Connect establishes a connection to PostgreSQL using the provided DSN and configures connection pooling
func Connect(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		databaseURL = "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// NewConnection initializes a database connection reading from DATABASE_URL
func NewConnection() (*gorm.DB, error) {
	return Connect("")
}

// AutoMigrate runs GORM auto-migrations for Venue, Event, and PricingTier models
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Venue{},
		&model.Event{},
		&model.PricingTier{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate database schema: %w", err)
	}
	return nil
}
