package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/lock"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestBookingService(t *testing.T) (*service.BookingService, *gorm.DB) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	if err := db.AutoMigrate(&model.Reservation{}, &model.ReservationItem{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	repo := repository.NewBookingRepository(db)
	locker := lock.NewDistributedLockManager(nil) // in-memory locker
	svc := service.NewBookingService(repo, locker, 5*time.Minute)

	return svc, db
}

func TestBookingService_Create_Confirm_Cancel(t *testing.T) {
	svc, _ := setupTestBookingService(t)
	ctx := context.Background()

	userID := uuid.New()
	eventID := uuid.New()
	seatID1 := uuid.New()
	seatID2 := uuid.New()

	req := dto.CreateReservationRequest{
		UserID:  userID,
		EventID: eventID,
		Seats: []dto.SeatItemDTO{
			{SeatID: seatID1, Section: "CAT 1 VIP", Row: "A", Number: "01", Price: 3500000},
			{SeatID: seatID2, Section: "CAT 1 VIP", Row: "A", Number: "02", Price: 3500000},
		},
		Currency: "IDR",
	}

	// 1. Create reservation
	res, err := svc.CreateReservation(ctx, req)
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}
	if res.Status != "LOCKED" {
		t.Errorf("expected status LOCKED, got %s", res.Status)
	}
	if res.TotalAmount != 7000000 {
		t.Errorf("expected total 7000000, got %f", res.TotalAmount)
	}

	// 2. Conflicting lock should fail
	_, errConflict := svc.CreateReservation(ctx, req)
	if errConflict == nil {
		t.Fatalf("expected conflict error on already locked seats, got nil")
	}

	// 3. Get reservation by ID
	fetched, err := svc.GetReservation(ctx, res.ReservationID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}
	if len(fetched.Seats) != 2 {
		t.Errorf("expected 2 seats, got %d", len(fetched.Seats))
	}

	// 4. Confirm reservation
	confirmed, err := svc.ConfirmReservation(ctx, res.ReservationID)
	if err != nil {
		t.Fatalf("ConfirmReservation failed: %v", err)
	}
	if confirmed.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", confirmed.Status)
	}
}

func TestBookingService_CancelReservation(t *testing.T) {
	svc, _ := setupTestBookingService(t)
	ctx := context.Background()

	req := dto.CreateReservationRequest{
		UserID:  uuid.New(),
		EventID: uuid.New(),
		Seats: []dto.SeatItemDTO{
			{SeatID: uuid.New(), Section: "CAT 2", Row: "B", Number: "10", Price: 2000000},
		},
		Currency: "IDR",
	}

	res, err := svc.CreateReservation(ctx, req)
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	// Cancel reservation
	err = svc.CancelReservation(ctx, res.ReservationID)
	if err != nil {
		t.Fatalf("CancelReservation failed: %v", err)
	}

	cancelled, err := svc.GetReservation(ctx, res.ReservationID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}
	if cancelled.Status != "CANCELLED" {
		t.Errorf("expected status CANCELLED, got %s", cancelled.Status)
	}
}
