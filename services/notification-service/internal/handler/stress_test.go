package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
	"github.com/gin-gonic/gin"
)

// TestHTTPHandler_QueueSaturationReturns503 tests that when the worker pool queue is saturated,
// concurrent incoming HTTP requests receive HTTP 503 Service Unavailable immediately without deadlocking.
func TestHTTPHandler_QueueSaturationReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSMTP, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockSMTP.Close()

	smtpClient, err := smtp.NewClient(
		"127.0.0.1",
		mockSMTP.Port(),
		"",
		"",
		"noreply@tickid.com",
		false,
	)
	if err != nil {
		t.Fatalf("failed to create smtp client: %v", err)
	}

	// Tiny pool of capacity 2, workers NOT started so the queue fills and stays full
	workerPool := queue.NewWorkerPool(1, 2)

	notifService, err := service.NewNotificationService(smtpClient, workerPool, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create notification service: %v", err)
	}

	notifHandler := handler.NewNotificationHandler(notifService)
	router := handler.SetupRouter(notifHandler, handler.NewHealthHandler())

	// Step 1: Fill the 2 slots with 202 Accepted
	for i := 0; i < 2; i++ {
		payload := fmt.Sprintf(`{"to": ["user%d@example.com"], "subject": "Hi", "body": "Msg"}`, i)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/send-custom", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusAccepted {
			t.Fatalf("slot %d: expected 202 Accepted, got %d", i, w.Code)
		}
	}

	// Step 2: Flood 50 concurrent requests into saturated queue
	const floodCount = 50
	var status503Count atomic.Int32
	var otherStatusCount atomic.Int32
	var blockedRequests atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < floodCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload := fmt.Sprintf(`{"to": ["flood%d@example.com"], "subject": "Hi", "body": "Msg"}`, idx)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/send-custom", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")

			done := make(chan struct{})
			w := httptest.NewRecorder()

			go func() {
				router.ServeHTTP(w, req)
				close(done)
			}()

			select {
			case <-done:
				if w.Code == http.StatusServiceUnavailable {
					status503Count.Add(1)
					var res dto.BaseResponse[any]
					_ = json.Unmarshal(w.Body.Bytes(), &res)
					if res.Success != false {
						t.Errorf("expected success=false on 503")
					}
				} else {
					otherStatusCount.Add(1)
				}
			case <-time.After(1 * time.Second):
				blockedRequests.Add(1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Saturation test results: 503 count = %d, other status = %d, blocked = %d",
		status503Count.Load(), otherStatusCount.Load(), blockedRequests.Load())

	if blockedRequests.Load() > 0 {
		t.Fatalf("%d HTTP requests hung/deadlocked during queue saturation!", blockedRequests.Load())
	}
	if status503Count.Load() != floodCount {
		t.Fatalf("expected all %d requests to return 503 Service Unavailable, got %d",
			floodCount, status503Count.Load())
	}
}
