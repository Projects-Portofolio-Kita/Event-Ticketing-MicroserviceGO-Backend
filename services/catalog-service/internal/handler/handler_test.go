package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, model.Venue, service.CacheService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to retrieve sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	_ = db.Exec("PRAGMA foreign_keys = ON;")
	if err := db.AutoMigrate(&model.Venue{}, &model.Event{}, &model.PricingTier{}); err != nil {
		t.Fatalf("failed to automigrate schema: %v", err)
	}

	t.Cleanup(func() {
		sqlDB.Close()
	})

	testVenue := model.Venue{
		ID:            uuid.New(),
		Name:          "Stadion Gelora Bung Karno",
		Address:       "Jl. Pintu Satu Senayan",
		City:          "Jakarta",
		TotalCapacity: 75000,
	}
	if err := db.Create(&testVenue).Error; err != nil {
		t.Fatalf("failed to seed test venue: %v", err)
	}

	eventRepo := repository.NewEventRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	tierRepo := repository.NewPricingTierRepository(db)

	cacheService := service.NewInMemoryCacheService(60 * time.Second)
	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	tierService := service.NewPricingTierService(tierRepo, eventRepo, cacheService)

	healthHandler := handler.NewHealthHandler()
	eventHandler := handler.NewEventHandler(eventService)
	tierHandler := handler.NewPricingTierHandler(tierService)

	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router, eventHandler, tierHandler, healthHandler)

	return router, db, testVenue, cacheService
}

func executeRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHealthCheck(t *testing.T) {
	router, _, _, _ := setupTestRouter(t)

	w := executeRequest(router, http.MethodGet, "/health", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	if resp["service"] != "catalog-service" {
		t.Errorf("expected service 'catalog-service', got %v", resp["service"])
	}
}

func TestCreateEvent_Success_And_Validation(t *testing.T) {
	router, _, testVenue, _ := setupTestRouter(t)

	t.Run("Create event successfully in DRAFT (201 Created)", func(t *testing.T) {
		reqPayload := dto.CreateEventRequest{
			Title:       "Coldplay Music of the Spheres",
			Description: "Live concert in Jakarta",
			VenueID:     testVenue.ID,
			EventDate:   time.Now().Add(48 * time.Hour),
		}

		w := executeRequest(router, http.MethodPost, "/api/v1/events", reqPayload)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.EventResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal BaseResponse: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success to be true")
		}
		if resp.Data.Status != model.StatusDraft {
			t.Errorf("expected status DRAFT, got %s", resp.Data.Status)
		}
		if resp.Data.Title != reqPayload.Title {
			t.Errorf("expected title %q, got %q", reqPayload.Title, resp.Data.Title)
		}
		if resp.Data.VenueID != testVenue.ID {
			t.Errorf("expected venue ID %s, got %s", testVenue.ID, resp.Data.VenueID)
		}
	})

	t.Run("Reject when venue does not exist (404 Not Found)", func(t *testing.T) {
		reqPayload := dto.CreateEventRequest{
			Title:       "Non-existent Venue Event",
			Description: "This should fail",
			VenueID:     uuid.New(),
			EventDate:   time.Now().Add(24 * time.Hour),
		}

		w := executeRequest(router, http.MethodPost, "/api/v1/events", reqPayload)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 Not Found, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Reject when title is missing (400 Bad Request)", func(t *testing.T) {
		reqPayload := map[string]interface{}{
			"description": "Missing title",
			"venue_id":    testVenue.ID.String(),
			"event_date":  time.Now().Add(24 * time.Hour),
		}

		w := executeRequest(router, http.MethodPost, "/api/v1/events", reqPayload)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestGetEventByID(t *testing.T) {
	router, db, testVenue, _ := setupTestRouter(t)

	// Create test event directly in DB
	event := model.Event{
		ID:          uuid.New(),
		Name:        "Ed Sheeran Mathematics Tour",
		Description: "Acoustic and stadium tour",
		VenueID:     testVenue.ID,
		Status:      model.StatusDraft,
		EventDate:   time.Now().Add(72 * time.Hour),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("failed to create test event: %v", err)
	}

	// Create a pricing tier for the event
	tier := model.PricingTier{
		ID:         uuid.New(),
		EventID:    event.ID,
		TierName:   "VIP Diamond",
		Price:      2500000,
		Currency:   "IDR",
		TotalSeats: 500,
	}
	if err := db.Create(&tier).Error; err != nil {
		t.Fatalf("failed to create test tier: %v", err)
	}

	t.Run("Retrieve event with nested venue and pricing tiers (200 OK)", func(t *testing.T) {
		w := executeRequest(router, http.MethodGet, "/api/v1/events/"+event.ID.String(), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.EventDetailResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal BaseResponse: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success true")
		}
		if resp.Data.ID != event.ID {
			t.Errorf("expected event ID %s, got %s", event.ID, resp.Data.ID)
		}
		if resp.Data.Venue.Name != testVenue.Name {
			t.Errorf("expected venue name %q, got %q", testVenue.Name, resp.Data.Venue.Name)
		}
		if len(resp.Data.PricingTiers) != 1 {
			t.Fatalf("expected 1 pricing tier, got %d", len(resp.Data.PricingTiers))
		}
		if resp.Data.PricingTiers[0].TierName != "VIP Diamond" {
			t.Errorf("expected tier name VIP Diamond, got %s", resp.Data.PricingTiers[0].TierName)
		}
	})

	t.Run("Return 404 for non-existent event ID", func(t *testing.T) {
		randomID := uuid.New().String()
		w := executeRequest(router, http.MethodGet, "/api/v1/events/"+randomID, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("Return 400 for invalid UUID format", func(t *testing.T) {
		w := executeRequest(router, http.MethodGet, "/api/v1/events/invalid-uuid-string", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestUpdateEventStatus_FSM_Transitions(t *testing.T) {
	router, db, testVenue, _ := setupTestRouter(t)

	// Create test event in DRAFT
	event := model.Event{
		ID:          uuid.New(),
		Name:        "Taylor Swift The Eras Tour",
		Description: "Pop concert",
		VenueID:     testVenue.ID,
		Status:      model.StatusDraft,
		EventDate:   time.Now().Add(100 * time.Hour),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	t.Run("Reject invalid transition DRAFT -> ON_SALE (422 Unprocessable Entity)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusOnSale,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422 Unprocessable Entity, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Allow valid transition DRAFT -> PUBLISHED (200 OK)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusPublished,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.EventResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if resp.Data.Status != model.StatusPublished {
			t.Errorf("expected status PUBLISHED, got %s", resp.Data.Status)
		}
	})

	t.Run("Allow valid transition PUBLISHED -> ON_SALE (200 OK)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusOnSale,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.EventResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.Status != model.StatusOnSale {
			t.Errorf("expected status ON_SALE, got %s", resp.Data.Status)
		}
	})

	t.Run("Allow valid transition ON_SALE -> SOLD_OUT (200 OK)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusSoldOut,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Allow valid transition SOLD_OUT -> ON_SALE (200 OK)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusOnSale,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Allow transition ON_SALE -> CANCELLED (200 OK)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusCancelled,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Reject transition from terminal CANCELLED state (422 Unprocessable Entity)", func(t *testing.T) {
		payload := dto.UpdateEventStatusRequest{
			Status: model.StatusPublished,
		}
		w := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", payload)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422 Unprocessable Entity, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestPricingTier_LockRule_And_Capacity(t *testing.T) {
	router, db, testVenue, _ := setupTestRouter(t)

	event := model.Event{
		ID:          uuid.New(),
		Name:        "Maroon 5 Tour",
		Description: "Pop rock",
		VenueID:     testVenue.ID,
		Status:      model.StatusDraft,
		EventDate:   time.Now().Add(120 * time.Hour),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	t.Run("Allow adding pricing tier when event is in DRAFT (201 Created)", func(t *testing.T) {
		tierPayload := dto.CreatePricingTierRequest{
			TierName:   "Festival GA",
			Price:      1500000,
			Currency:   "IDR",
			TotalSeats: 3000,
		}
		w := executeRequest(router, http.MethodPost, "/api/v1/events/"+event.ID.String()+"/tiers", tierPayload)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.PricingTierResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Data.TierName != "Festival GA" {
			t.Errorf("expected tier name 'Festival GA', got %s", resp.Data.TierName)
		}
		if resp.Data.TotalSeats != 3000 {
			t.Errorf("expected 3000 seats, got %d", resp.Data.TotalSeats)
		}
	})

	t.Run("Allow adding pricing tier when event is in PUBLISHED (201 Created)", func(t *testing.T) {
		// Transition event to PUBLISHED
		db.Model(&event).Update("status", model.StatusPublished)

		tierPayload := dto.CreatePricingTierRequest{
			TierName:   "Tribune East",
			Price:      950000,
			Currency:   "IDR",
			TotalSeats: 2000,
		}
		w := executeRequest(router, http.MethodPost, "/api/v1/events/"+event.ID.String()+"/tiers", tierPayload)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Strictly reject adding pricing tier when event is ON_SALE (409 Conflict)", func(t *testing.T) {
		// Transition event to ON_SALE
		db.Model(&event).Update("status", model.StatusOnSale)

		tierPayload := dto.CreatePricingTierRequest{
			TierName:   "Late Bird VIP",
			Price:      2000000,
			Currency:   "IDR",
			TotalSeats: 500,
		}
		w := executeRequest(router, http.MethodPost, "/api/v1/events/"+event.ID.String()+"/tiers", tierPayload)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Strictly reject adding pricing tier when event is SOLD_OUT (409 Conflict)", func(t *testing.T) {
		// Transition event to SOLD_OUT
		db.Model(&event).Update("status", model.StatusSoldOut)

		tierPayload := dto.CreatePricingTierRequest{
			TierName:   "Extra Seats",
			Price:      1000000,
			Currency:   "IDR",
			TotalSeats: 100,
		}
		w := executeRequest(router, http.MethodPost, "/api/v1/events/"+event.ID.String()+"/tiers", tierPayload)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetPricingTiers returns all added tiers (200 OK)", func(t *testing.T) {
		w := executeRequest(router, http.MethodGet, "/api/v1/events/"+event.ID.String()+"/tiers", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[[]dto.PricingTierResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode pricing tiers response: %v", err)
		}
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 tiers, got %d", len(resp.Data))
		}
	})
}

func TestSearchEvents_Pagination_And_Filtering(t *testing.T) {
	router, db, testVenue, _ := setupTestRouter(t)

	// Create second venue in Tangerang
	venue2 := model.Venue{
		ID:            uuid.New(),
		Name:          "ICE BSD",
		Address:       "BSD City",
		City:          "Tangerang",
		TotalCapacity: 10000,
	}
	db.Create(&venue2)

	now := time.Now()

	// Seed 3 distinct events
	e1 := model.Event{
		ID:          uuid.New(),
		Name:        "Rock Festival 2026",
		Description: "Annual rock fest",
		VenueID:     testVenue.ID,
		Status:      model.StatusPublished,
		EventDate:   now.Add(24 * time.Hour),
	}
	e2 := model.Event{
		ID:          uuid.New(),
		Name:        "Jazz in the City",
		Description: "Smooth jazz night",
		VenueID:     testVenue.ID,
		Status:      model.StatusDraft,
		EventDate:   now.Add(48 * time.Hour),
	}
	e3 := model.Event{
		ID:          uuid.New(),
		Name:        "K-Pop Super Concert",
		Description: "Major Korean artists",
		VenueID:     venue2.ID,
		Status:      model.StatusPublished,
		EventDate:   now.Add(72 * time.Hour),
	}
	db.Create(&e1)
	db.Create(&e2)
	db.Create(&e3)

	t.Run("Query all events with pagination", func(t *testing.T) {
		w := executeRequest(router, http.MethodGet, "/api/v1/events?page=1&limit=2", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.PaginatedEventsResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(resp.Data.Items))
		}
		if resp.Data.Meta.TotalItems != 3 {
			t.Errorf("expected 3 total items, got %d", resp.Data.Meta.TotalItems)
		}
		if resp.Data.Meta.TotalPages != 2 {
			t.Errorf("expected 2 total pages, got %d", resp.Data.Meta.TotalPages)
		}
	})

	t.Run("Filter events by status=PUBLISHED", func(t *testing.T) {
		w := executeRequest(router, http.MethodGet, "/api/v1/events?status=PUBLISHED", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.PaginatedEventsResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Data.Items) != 2 {
			t.Errorf("expected 2 published events, got %d", len(resp.Data.Items))
		}
		for _, it := range resp.Data.Items {
			if it.Status != model.StatusPublished {
				t.Errorf("expected status PUBLISHED, got %s", it.Status)
			}
		}
	})

	t.Run("Filter events by city=Tangerang", func(t *testing.T) {
		w := executeRequest(router, http.MethodGet, "/api/v1/events?city=Tangerang", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.PaginatedEventsResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Data.Items) != 1 {
			t.Fatalf("expected 1 event in Tangerang, got %d", len(resp.Data.Items))
		}
		if resp.Data.Items[0].Title != "K-Pop Super Concert" {
			t.Errorf("expected 'K-Pop Super Concert', got %q", resp.Data.Items[0].Title)
		}
	})

	t.Run("Filter events by date range", func(t *testing.T) {
		startDate := now.Add(36 * time.Hour).UTC().Format(time.RFC3339)
		endDate := now.Add(60 * time.Hour).UTC().Format(time.RFC3339)

		url := fmt.Sprintf("/api/v1/events?start_date=%s&end_date=%s", startDate, endDate)
		w := executeRequest(router, http.MethodGet, url, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp dto.BaseResponse[dto.PaginatedEventsResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Data.Items) != 1 {
			t.Fatalf("expected 1 event in range, got %d", len(resp.Data.Items))
		}
		if resp.Data.Items[0].Title != "Jazz in the City" {
			t.Errorf("expected 'Jazz in the City', got %q", resp.Data.Items[0].Title)
		}
	})
}

func TestCache_HitMiss_And_MutationEviction(t *testing.T) {
	router, db, testVenue, cacheService := setupTestRouter(t)

	event := model.Event{
		ID:          uuid.New(),
		Name:        "Bruno Mars Live",
		Description: "Pop performance",
		VenueID:     testVenue.ID,
		Status:      model.StatusDraft,
		EventDate:   time.Now().Add(48 * time.Hour),
	}
	db.Create(&event)

	// 1. Initial GET /api/v1/events/:id populates cache
	w1 := executeRequest(router, http.MethodGet, "/api/v1/events/"+event.ID.String(), nil)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w1.Code)
	}

	// Verify key is in cache
	eventCacheKey := "catalog:event:" + event.ID.String()
	var cachedDetail dto.EventDetailResponse
	if !cacheService.Get(context.Background(), eventCacheKey, &cachedDetail) {
		t.Fatalf("expected cache key %q to be populated", eventCacheKey)
	}

	// Directly modify database name behind cache
	db.Model(&event).Update("name", "Bruno Mars Live (Direct DB Edit)")

	// 2. Second GET should hit cache and still return original name
	w2 := executeRequest(router, http.MethodGet, "/api/v1/events/"+event.ID.String(), nil)
	var resp2 dto.BaseResponse[dto.EventDetailResponse]
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Data.Title != "Bruno Mars Live" {
		t.Errorf("expected cached name 'Bruno Mars Live', got %q", resp2.Data.Title)
	}

	// 3. Mutate event status via PATCH endpoint -> should evict cache
	statusPayload := dto.UpdateEventStatusRequest{
		Status: model.StatusPublished,
	}
	w3 := executeRequest(router, http.MethodPatch, "/api/v1/events/"+event.ID.String()+"/status", statusPayload)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w3.Code)
	}

	// Verify cache key was evicted
	if cacheService.Get(context.Background(), eventCacheKey, &cachedDetail) {
		t.Fatalf("expected cache key %q to be invalidated after status mutation", eventCacheKey)
	}

	// 4. Third GET queries fresh database data
	w4 := executeRequest(router, http.MethodGet, "/api/v1/events/"+event.ID.String(), nil)
	var resp4 dto.BaseResponse[dto.EventDetailResponse]
	_ = json.Unmarshal(w4.Body.Bytes(), &resp4)
	if resp4.Data.Status != model.StatusPublished {
		t.Errorf("expected fresh status PUBLISHED, got %s", resp4.Data.Status)
	}
}
