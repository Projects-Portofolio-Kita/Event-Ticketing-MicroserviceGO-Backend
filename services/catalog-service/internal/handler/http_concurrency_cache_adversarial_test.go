package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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

// setupConcurrencyTestRouter initializes an isolated in-memory router with SQLite & InMemoryCache
func setupConcurrencyTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, model.Venue, service.CacheService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
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
	_ = db.Exec("PRAGMA busy_timeout = 5000;")
	if err := db.AutoMigrate(&model.Venue{}, &model.Event{}, &model.PricingTier{}); err != nil {
		t.Fatalf("failed to automigrate schema: %v", err)
	}

	t.Cleanup(func() {
		sqlDB.Close()
	})

	testVenue := model.Venue{
		ID:            uuid.New(),
		Name:          "Gelora Bung Karno Main Stadium",
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

func doHTTPReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
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

// ============================================================================
// 1. EMPIRICAL VERIFICATION: ON_SALE & SOLD_OUT 409 CONFLICT UNDER CONCURRENCY
// ============================================================================

func TestAdversarial_ConcurrentTierCreation_OnSale_Returns409Conflict(t *testing.T) {
	router, db, testVenue, _ := setupConcurrencyTestRouter(t)

	// Create event and transition DRAFT -> PUBLISHED -> ON_SALE
	event := model.Event{
		ID:          uuid.New(),
		Name:        "High-Demand Stadium Rock Concert",
		Description: "Rapid ticketing stress test",
		VenueID:     testVenue.ID,
		Status:      model.StatusDraft,
		EventDate:   time.Now().Add(72 * time.Hour),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("failed to seed event: %v", err)
	}

	// Move to PUBLISHED
	wPub := doHTTPReq(router, http.MethodPatch, fmt.Sprintf("/api/v1/events/%s/status", event.ID), dto.UpdateEventStatusRequest{
		Status: model.StatusPublished,
	})
	if wPub.Code != http.StatusOK {
		t.Fatalf("failed to transition to PUBLISHED: %d %s", wPub.Code, wPub.Body.String())
	}

	// Move to ON_SALE
	wSale := doHTTPReq(router, http.MethodPatch, fmt.Sprintf("/api/v1/events/%s/status", event.ID), dto.UpdateEventStatusRequest{
		Status: model.StatusOnSale,
	})
	if wSale.Code != http.StatusOK {
		t.Fatalf("failed to transition to ON_SALE: %d %s", wSale.Code, wSale.Body.String())
	}

	const concurrency = 100
	var wg sync.WaitGroup
	var conflictCount int64
	var otherCount int64

	startSignal := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startSignal

			tierPayload := dto.CreatePricingTierRequest{
				TierName:   fmt.Sprintf("Adversarial Tier %d", workerID),
				Price:      1000000 + float64(workerID*10000),
				Currency:   "IDR",
				TotalSeats: 100,
			}

			w := doHTTPReq(router, http.MethodPost, fmt.Sprintf("/api/v1/events/%s/tiers", event.ID), tierPayload)

			if w.Code == http.StatusConflict {
				atomic.AddInt64(&conflictCount, 1)

				var errResp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err == nil {
					if errResp["success"] == true {
						t.Errorf("worker %d: expected success=false in conflict response", workerID)
					}
				}
			} else {
				atomic.AddInt64(&otherCount, 1)
				t.Errorf("worker %d: expected 409 Conflict, got HTTP %d: %s", workerID, w.Code, w.Body.String())
			}
		}(i)
	}

	// Unleash all 100 goroutines simultaneously
	close(startSignal)
	wg.Wait()

	t.Logf("Empirical Result: %d requests -> 409 Conflict: %d, Other Statuses: %d", concurrency, conflictCount, otherCount)

	if conflictCount != int64(concurrency) {
		t.Fatalf("Expected 100%% (%d) of rapid tier addition requests on ON_SALE to return 409 Conflict, got %d", concurrency, conflictCount)
	}
	if otherCount != 0 {
		t.Fatalf("Expected 0 non-409 responses, got %d", otherCount)
	}

	// Verify database integrity: zero tiers were added
	var tierCount int64
	if err := db.Model(&model.PricingTier{}).Where("event_id = ?", event.ID).Count(&tierCount).Error; err != nil {
		t.Fatalf("failed to count tiers: %v", err)
	}
	if tierCount != 0 {
		t.Fatalf("CRITICAL INTEGRITY FAILURE: Expected 0 tiers in DB, but found %d tiers inserted!", tierCount)
	}
}

func TestAdversarial_ConcurrentTierCreation_SoldOut_Returns409Conflict(t *testing.T) {
	router, db, testVenue, _ := setupConcurrencyTestRouter(t)

	// Event moved through: DRAFT -> PUBLISHED -> ON_SALE -> SOLD_OUT
	event := model.Event{
		ID:        uuid.New(),
		Name:      "Sold Out Festival",
		VenueID:   testVenue.ID,
		Status:    model.StatusDraft,
		EventDate: time.Now().Add(48 * time.Hour),
	}
	db.Create(&event)
	db.Model(&event).Update("status", model.StatusSoldOut)

	const concurrency = 50
	var wg sync.WaitGroup
	var conflictCount int64

	startSignal := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startSignal

			tierPayload := dto.CreatePricingTierRequest{
				TierName:   fmt.Sprintf("Scalper Tier %d", workerID),
				Price:      5000000,
				Currency:   "IDR",
				TotalSeats: 10,
			}

			w := doHTTPReq(router, http.MethodPost, fmt.Sprintf("/api/v1/events/%s/tiers", event.ID), tierPayload)
			if w.Code == http.StatusConflict {
				atomic.AddInt64(&conflictCount, 1)
			}
		}(i)
	}

	close(startSignal)
	wg.Wait()

	if conflictCount != int64(concurrency) {
		t.Fatalf("Expected all %d requests on SOLD_OUT event to return 409 Conflict, got %d", concurrency, conflictCount)
	}

	var tierCount int64
	db.Model(&model.PricingTier{}).Where("event_id = ?", event.ID).Count(&tierCount)
	if tierCount != 0 {
		t.Fatalf("Expected 0 tiers in DB for SOLD_OUT event, found %d", tierCount)
	}
}

// ============================================================================
// 2. EMPIRICAL VERIFICATION: RACE BETWEEN TIER CREATION AND ON_SALE TRANSITION
// ============================================================================

func TestAdversarial_Race_TiersVsStatusTransitionToOnSale(t *testing.T) {
	router, db, testVenue, _ := setupConcurrencyTestRouter(t)

	event := model.Event{
		ID:        uuid.New(),
		Name:      "Race Condition Tour",
		VenueID:   testVenue.ID,
		Status:    model.StatusPublished, // Starts in PUBLISHED (tier addition allowed)
		EventDate: time.Now().Add(120 * time.Hour),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	const tierWorkers = 50
	var wg sync.WaitGroup
	var createdCount int64
	var conflictCount int64
	var otherCount int64

	startSignal := make(chan struct{})

	// Group A: 50 concurrent workers attempting to add tiers
	for i := 0; i < tierWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startSignal

			tierPayload := dto.CreatePricingTierRequest{
				TierName:   fmt.Sprintf("Race Tier %02d", workerID),
				Price:      750000,
				Currency:   "IDR",
				TotalSeats: 500,
			}

			w := doHTTPReq(router, http.MethodPost, fmt.Sprintf("/api/v1/events/%s/tiers", event.ID), tierPayload)

			switch w.Code {
			case http.StatusCreated:
				atomic.AddInt64(&createdCount, 1)
			case http.StatusConflict:
				atomic.AddInt64(&conflictCount, 1)
			default:
				atomic.AddInt64(&otherCount, 1)
				t.Errorf("Unexpected status code %d: %s", w.Code, w.Body.String())
			}
		}(i)
	}

	// Group B: 1 concurrent worker attempting to transition event to ON_SALE
	var statusUpdated int64
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startSignal

		w := doHTTPReq(router, http.MethodPatch, fmt.Sprintf("/api/v1/events/%s/status", event.ID), dto.UpdateEventStatusRequest{
			Status: model.StatusOnSale,
		})
		if w.Code == http.StatusOK {
			atomic.StoreInt64(&statusUpdated, 1)
		}
	}()

	close(startSignal)
	wg.Wait()

	t.Logf("Race Results: Created=%d, 409 Conflict=%d, Other=%d, StatusUpdated=%d",
		createdCount, conflictCount, otherCount, statusUpdated)

	if otherCount > 0 {
		t.Fatalf("Encountered %d unexpected error responses (e.g. 500 or corruptions)", otherCount)
	}

	if atomic.LoadInt64(&statusUpdated) != 1 {
		t.Fatalf("Expected status transition to ON_SALE to succeed")
	}

	if createdCount+conflictCount != int64(tierWorkers) {
		t.Fatalf("Total tier worker responses (%d + %d) did not equal %d", createdCount, conflictCount, tierWorkers)
	}

	// Verify database consistency: The number of tiers in DB MUST EXACTLY equal createdCount
	var actualTiersInDB int64
	db.Model(&model.PricingTier{}).Where("event_id = ?", event.ID).Count(&actualTiersInDB)
	if actualTiersInDB != createdCount {
		t.Fatalf("Database inconsistency: createdCount=%d but actual DB rows=%d", createdCount, actualTiersInDB)
	}

	// Verify final event status in DB is ON_SALE
	var finalEvent model.Event
	db.First(&finalEvent, "id = ?", event.ID)
	if finalEvent.Status != model.StatusOnSale {
		t.Fatalf("Expected final event status to be ON_SALE, got: %s", finalEvent.Status)
	}
}

// ============================================================================
// 3. EMPIRICAL VERIFICATION: HIGH-CONCURRENCY HTTP READS
// ============================================================================

func TestAdversarial_ConcurrentHTTPReads_NoDataRaces(t *testing.T) {
	router, db, testVenue, _ := setupConcurrencyTestRouter(t)

	// Seed event with 3 tiers
	event := model.Event{
		ID:        uuid.New(),
		Name:      "Concurrent Read Fest",
		VenueID:   testVenue.ID,
		Status:    model.StatusPublished,
		EventDate: time.Now().Add(24 * time.Hour),
	}
	db.Create(&event)

	tiers := []model.PricingTier{
		{ID: uuid.New(), EventID: event.ID, TierName: "VVIP", Price: 3000000, Currency: "IDR", TotalSeats: 100},
		{ID: uuid.New(), EventID: event.ID, TierName: "VIP", Price: 2000000, Currency: "IDR", TotalSeats: 300},
		{ID: uuid.New(), EventID: event.ID, TierName: "General", Price: 1000000, Currency: "IDR", TotalSeats: 1000},
	}
	for _, tier := range tiers {
		db.Create(&tier)
	}

	const totalRequests = 150
	var wg sync.WaitGroup
	var successCount int64

	startSignal := make(chan struct{})

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startSignal

			var path string
			switch workerID % 3 {
			case 0:
				path = "/api/v1/events"
			case 1:
				path = fmt.Sprintf("/api/v1/events/%s", event.ID)
			case 2:
				path = fmt.Sprintf("/api/v1/events/%s/tiers", event.ID)
			}

			w := doHTTPReq(router, http.MethodGet, path, nil)
			if w.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				t.Errorf("worker %d: GET %s failed with status %d: %s", workerID, path, w.Code, w.Body.String())
			}
		}(i)
	}

	close(startSignal)
	wg.Wait()

	if successCount != int64(totalRequests) {
		t.Fatalf("Expected %d successful reads, got %d", totalRequests, successCount)
	}
}

// ============================================================================
// 4. EMPIRICAL VERIFICATION: CACHE INVALIDATION ON STATUS AND TIER MUTATIONS
// ============================================================================

func TestAdversarial_CacheInvalidation_StatusMutationEvictsEventAndCatalog(t *testing.T) {
	router, db, testVenue, cacheService := setupConcurrencyTestRouter(t)
	ctx := context.Background()

	// Create 2 events
	eventA := model.Event{
		ID:        uuid.New(),
		Name:      "Event Alpha",
		VenueID:   testVenue.ID,
		Status:    model.StatusDraft,
		EventDate: time.Now().Add(24 * time.Hour),
	}
	eventB := model.Event{
		ID:        uuid.New(),
		Name:      "Event Beta",
		VenueID:   testVenue.ID,
		Status:    model.StatusDraft,
		EventDate: time.Now().Add(48 * time.Hour),
	}
	db.Create(&eventA)
	db.Create(&eventB)

	// Step 1: Initial GET /api/v1/events/:id populates cache
	wDetail1 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", eventA.ID), nil)
	if wDetail1.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", wDetail1.Code)
	}

	eventCacheKey := service.EventKeyPrefix + eventA.ID.String()
	var cachedDetail dto.EventDetailResponse
	if !cacheService.Get(ctx, eventCacheKey, &cachedDetail) {
		t.Fatalf("expected cache key %q to be populated", eventCacheKey)
	}
	if cachedDetail.Status != model.StatusDraft {
		t.Fatalf("expected cached status DRAFT, got: %s", cachedDetail.Status)
	}

	// Step 2: Initial GET /api/v1/events?status=DRAFT populates catalog search cache
	wListDraft1 := doHTTPReq(router, http.MethodGet, "/api/v1/events?status=DRAFT", nil)
	if wListDraft1.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", wListDraft1.Code)
	}

	var draftResp1 dto.BaseResponse[dto.PaginatedEventsResponse]
	_ = json.Unmarshal(wListDraft1.Body.Bytes(), &draftResp1)
	if len(draftResp1.Data.Items) != 2 {
		t.Fatalf("expected 2 draft events, got: %d", len(draftResp1.Data.Items))
	}

	// Step 3: Mutate DB directly behind the cache to prove cache HIT behavior
	db.Model(&eventA).Update("name", "Event Alpha Ghost Modified Name")

	// Call GET again: MUST return cached original name, NOT ghost name
	wDetail2 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", eventA.ID), nil)
	var detailResp2 dto.BaseResponse[dto.EventDetailResponse]
	_ = json.Unmarshal(wDetail2.Body.Bytes(), &detailResp2)
	if detailResp2.Data.Title != "Event Alpha" {
		t.Fatalf("expected cached title 'Event Alpha', got ghost title: %s", detailResp2.Data.Title)
	}

	// Step 4: Perform Status Mutation via PATCH /api/v1/events/:id/status
	wPatch := doHTTPReq(router, http.MethodPatch, fmt.Sprintf("/api/v1/events/%s/status", eventA.ID), dto.UpdateEventStatusRequest{
		Status: model.StatusPublished,
	})
	if wPatch.Code != http.StatusOK {
		t.Fatalf("PATCH status failed: %d %s", wPatch.Code, wPatch.Body.String())
	}

	// Step 5: Verify cache eviction!
	if cacheService.Get(ctx, eventCacheKey, &cachedDetail) {
		t.Fatalf("CRITICAL CACHE BUG: Cache key %q was NOT evicted after status mutation!", eventCacheKey)
	}

	// Step 6: Subsequent GET /api/v1/events/:id MUST fetch fresh DB data
	wDetail3 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", eventA.ID), nil)
	var detailResp3 dto.BaseResponse[dto.EventDetailResponse]
	_ = json.Unmarshal(wDetail3.Body.Bytes(), &detailResp3)
	if detailResp3.Data.Status != model.StatusPublished {
		t.Fatalf("expected fresh status PUBLISHED, got: %s", detailResp3.Data.Status)
	}
	if detailResp3.Data.Title != "Event Alpha Ghost Modified Name" {
		t.Fatalf("expected fresh title from DB 'Event Alpha Ghost Modified Name', got: %s", detailResp3.Data.Title)
	}

	// Step 7: Verify catalog search cache was also evicted!
	wListDraft2 := doHTTPReq(router, http.MethodGet, "/api/v1/events?status=DRAFT", nil)
	var draftResp2 dto.BaseResponse[dto.PaginatedEventsResponse]
	_ = json.Unmarshal(wListDraft2.Body.Bytes(), &draftResp2)
	if len(draftResp2.Data.Items) != 1 {
		t.Fatalf("expected only 1 event in DRAFT after invalidation, got: %d", len(draftResp2.Data.Items))
	}
	if draftResp2.Data.Items[0].ID != eventB.ID {
		t.Fatalf("expected remaining draft event to be Event B (%s), got: %s", eventB.ID, draftResp2.Data.Items[0].ID)
	}

	// Step 8: GET /api/v1/events?status=PUBLISHED now returns Event Alpha
	wListPub := doHTTPReq(router, http.MethodGet, "/api/v1/events?status=PUBLISHED", nil)
	var pubResp dto.BaseResponse[dto.PaginatedEventsResponse]
	_ = json.Unmarshal(wListPub.Body.Bytes(), &pubResp)
	if len(pubResp.Data.Items) != 1 {
		t.Fatalf("expected 1 published event, got: %d", len(pubResp.Data.Items))
	}
	if pubResp.Data.Items[0].ID != eventA.ID {
		t.Fatalf("expected published event to be Event A (%s), got: %s", eventA.ID, pubResp.Data.Items[0].ID)
	}
}

func TestAdversarial_CacheInvalidation_TierMutationEvictsEventAndTiersAndCatalog(t *testing.T) {
	router, db, testVenue, cacheService := setupConcurrencyTestRouter(t)
	ctx := context.Background()

	event := model.Event{
		ID:        uuid.New(),
		Name:      "Symphony Orchestra",
		VenueID:   testVenue.ID,
		Status:    model.StatusPublished,
		EventDate: time.Now().Add(96 * time.Hour),
	}
	db.Create(&event)

	// Step 1: Pre-populate event detail cache (currently 0 tiers)
	wDetail1 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", event.ID), nil)
	if wDetail1.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", wDetail1.Code)
	}

	var detailResp1 dto.BaseResponse[dto.EventDetailResponse]
	_ = json.Unmarshal(wDetail1.Body.Bytes(), &detailResp1)
	if len(detailResp1.Data.PricingTiers) != 0 {
		t.Fatalf("expected 0 initial tiers, got: %d", len(detailResp1.Data.PricingTiers))
	}

	// Pre-populate tiers list cache
	wTiers1 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s/tiers", event.ID), nil)
	if wTiers1.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", wTiers1.Code)
	}

	eventCacheKey := service.EventKeyPrefix + event.ID.String()
	tiersCacheKey := service.EventKeyPrefix + event.ID.String() + ":tiers"

	var cachedDetail dto.EventDetailResponse
	var cachedTiers []dto.PricingTierResponse
	if !cacheService.Get(ctx, eventCacheKey, &cachedDetail) {
		t.Fatalf("expected event detail cache to be present")
	}
	if !cacheService.Get(ctx, tiersCacheKey, &cachedTiers) {
		t.Fatalf("expected tiers cache to be present")
	}

	// Pre-populate catalog search cache
	wCatalog1 := doHTTPReq(router, http.MethodGet, "/api/v1/events", nil)
	if wCatalog1.Code != http.StatusOK {
		t.Fatalf("expected 200, got: %d", wCatalog1.Code)
	}

	// Step 2: Mutate by adding a tier via HTTP endpoint
	newTierReq := dto.CreatePricingTierRequest{
		TierName:   "Concert Hall Balcony",
		Price:      1250000,
		Currency:   "IDR",
		TotalSeats: 250,
	}
	wAddTier := doHTTPReq(router, http.MethodPost, fmt.Sprintf("/api/v1/events/%s/tiers", event.ID), newTierReq)
	if wAddTier.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for tier addition, got: %d %s", wAddTier.Code, wAddTier.Body.String())
	}

	// Step 3: Verify cache keys were evicted
	if cacheService.Get(ctx, eventCacheKey, &cachedDetail) {
		t.Fatalf("CRITICAL CACHE BUG: event detail key %q was NOT evicted after tier addition!", eventCacheKey)
	}
	if cacheService.Get(ctx, tiersCacheKey, &cachedTiers) {
		t.Fatalf("CRITICAL CACHE BUG: event tiers key %q was NOT evicted after tier addition!", tiersCacheKey)
	}

	// Step 4: Subsequent GET /api/v1/events/:id must reflect the new tier
	wDetail2 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", event.ID), nil)
	var detailResp2 dto.BaseResponse[dto.EventDetailResponse]
	_ = json.Unmarshal(wDetail2.Body.Bytes(), &detailResp2)
	if len(detailResp2.Data.PricingTiers) != 1 {
		t.Fatalf("expected 1 tier in fresh event detail, got: %d", len(detailResp2.Data.PricingTiers))
	}
	if detailResp2.Data.PricingTiers[0].TierName != "Concert Hall Balcony" {
		t.Fatalf("expected tier 'Concert Hall Balcony', got: %s", detailResp2.Data.PricingTiers[0].TierName)
	}

	// Step 5: Subsequent GET /api/v1/events/:id/tiers must reflect the new tier
	wTiers2 := doHTTPReq(router, http.MethodGet, fmt.Sprintf("/api/v1/events/%s/tiers", event.ID), nil)
	var tiersResp2 dto.BaseResponse[[]dto.PricingTierResponse]
	_ = json.Unmarshal(wTiers2.Body.Bytes(), &tiersResp2)
	if len(tiersResp2.Data) != 1 {
		t.Fatalf("expected 1 tier in fresh tiers response, got: %d", len(tiersResp2.Data))
	}
	if tiersResp2.Data[0].TierName != "Concert Hall Balcony" {
		t.Fatalf("expected tier 'Concert Hall Balcony', got: %s", tiersResp2.Data[0].TierName)
	}
}
