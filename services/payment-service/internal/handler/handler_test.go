package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *service.PaymentService, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	if err := db.AutoMigrate(&model.Payment{}); err != nil {
		t.Fatalf("failed to automigrate payment schema: %v", err)
	}

	repo := repository.NewPaymentRepository(db)
	qris := provider.NewMockQRISProvider()
	booking := service.NewMockBookingClient()
	notif := service.NewMockNotificationClient()
	svc := service.NewPaymentService(repo, qris, booking, notif)
	h := handler.NewPaymentHandler(svc)

	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/payments", h.CreatePayment)
		api.GET("/payments/:id", h.GetPayment)
		api.POST("/payments/:id/confirm", h.ConfirmPayment)
		api.POST("/payments/:id/simulate-success", h.ConfirmPayment)
	}

	return router, svc, db
}

func TestHandler_CreatePayment_Success(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	resID := uuid.New()

	reqBody := dto.CreatePaymentRequest{
		ReservationID: resID.String(),
		PaymentMethod: "QRIS",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["qr_string"] == "" || resp["qr_string"] == nil {
		t.Errorf("expected non-empty qr_string in response")
	}
	if resp["status"] != "PENDING" {
		t.Errorf("expected status PENDING, got %v", resp["status"])
	}
}

func TestHandler_CreatePayment_CaseInsensitiveMethod(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	resID := uuid.New()

	// lowercase "qris" should be accepted cleanly
	reqBody := dto.CreatePaymentRequest{
		ReservationID: resID.String(),
		PaymentMethod: "qris",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created for lowercase 'qris', got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ConfirmPayment_Flow(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	resID := uuid.New()

	// 1. Create payment
	createReq := dto.CreatePaymentRequest{
		ReservationID: resID.String(),
		PaymentMethod: "QRIS",
	}
	bodyBytes, _ := json.Marshal(createReq)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)

	var created map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &created)
	payID := created["id"].(string)

	// 2. Confirm payment
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/payments/"+payID+"/confirm", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for payment confirm, got %d: %s", w2.Code, w2.Body.String())
	}

	var confirmed map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &confirmed)
	if confirmed["status"] != "SUCCEEDED" {
		t.Errorf("expected status SUCCEEDED, got %v", confirmed["status"])
	}

	// 3. GET payment to verify persistence
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/"+payID, nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for get payment, got %d", w3.Code)
	}
	var fetched map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &fetched)
	if fetched["status"] != "SUCCEEDED" {
		t.Errorf("expected persisted status SUCCEEDED, got %v", fetched["status"])
	}
}
