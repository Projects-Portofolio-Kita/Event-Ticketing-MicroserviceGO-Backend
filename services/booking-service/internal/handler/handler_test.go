package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/lock"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestBookingRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

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
	locker := lock.NewDistributedLockManager(nil)
	svc := service.NewBookingService(repo, locker, 5*time.Minute)
	h := handler.NewBookingHandler(svc)

	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/reservations", h.CreateReservation)
		api.GET("/reservations/:id", h.GetReservation)
		api.POST("/reservations/:id/confirm", h.ConfirmReservation)
		api.POST("/reservations/:id/cancel", h.CancelReservation)
	}

	return router, db
}

func TestBookingHandler_FullLifecycle(t *testing.T) {
	router, _ := setupTestBookingRouter(t)

	// 1. POST /api/v1/reservations
	createReq := dto.CreateReservationRequest{
		UserID:  uuid.New(),
		EventID: uuid.New(),
		Seats: []dto.SeatItemDTO{
			{SeatID: uuid.New(), Section: "CAT 1 VIP", Row: "C", Number: "14", Price: 3500000},
			{SeatID: uuid.New(), Section: "CAT 1 VIP", Row: "C", Number: "15", Price: 3500000},
		},
		Currency: "IDR",
	}
	body, _ := json.Marshal(createReq)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w1.Code, w1.Body.String())
	}

	var res dto.ReservationResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse reservation response: %v", err)
	}
	if res.Status != "LOCKED" {
		t.Errorf("expected status LOCKED, got %s", res.Status)
	}

	// 2. GET /api/v1/reservations/:id
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/reservations/"+res.ReservationID.String(), nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w2.Code)
	}

	// 3. POST /api/v1/reservations/:id/confirm
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/api/v1/reservations/"+res.ReservationID.String()+"/confirm", nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on confirm, got %d: %s", w3.Code, w3.Body.String())
	}

	var confRes dto.ReservationResponse
	json.Unmarshal(w3.Body.Bytes(), &confRes)
	if confRes.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", confRes.Status)
	}
}
