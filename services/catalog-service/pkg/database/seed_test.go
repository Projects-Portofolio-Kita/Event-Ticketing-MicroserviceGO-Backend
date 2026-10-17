package database_test

import (
	"sync"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&model.Venue{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	t.Cleanup(func() {
		sqlDB.Close()
	})
	return db
}

func TestSeedVenues_NilDB(t *testing.T) {
	err := database.SeedVenues(nil)
	if err == nil {
		t.Fatal("expected error when db is nil, but got nil")
	}
	expected := "db connection is required"
	if err.Error() != expected {
		t.Fatalf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestSeedVenues_Idempotency(t *testing.T) {
	db := setupSQLiteDB(t)

	// First call
	if err := database.SeedVenues(db); err != nil {
		t.Fatalf("first SeedVenues call failed: %v", err)
	}

	var count1 int64
	if err := db.Model(&model.Venue{}).Count(&count1).Error; err != nil {
		t.Fatalf("failed to count venues: %v", err)
	}
	if count1 != 3 {
		t.Fatalf("expected exactly 3 venues after first seed, got %d", count1)
	}

	// Second call (idempotency test)
	if err := database.SeedVenues(db); err != nil {
		t.Fatalf("second SeedVenues call failed: %v", err)
	}

	var count2 int64
	if err := db.Model(&model.Venue{}).Count(&count2).Error; err != nil {
		t.Fatalf("failed to count venues: %v", err)
	}
	if count2 != 3 {
		t.Fatalf("expected venue count to remain 3 after second seed, got %d", count2)
	}

	// Concurrent multi-threaded execution test
	const concurrency = 10
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := database.SeedVenues(db); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent SeedVenues failed: %v", err)
	}

	var countFinal int64
	if err := db.Model(&model.Venue{}).Count(&countFinal).Error; err != nil {
		t.Fatalf("failed to count venues after concurrent execution: %v", err)
	}
	if countFinal != 3 {
		t.Fatalf("expected final venue count to remain 3 after concurrent calls, got %d", countFinal)
	}
}
