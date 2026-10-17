package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	pkgredis "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/redis"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestEnvironment spins up SQLite in-memory and initializes repository implementations
func setupTestEnvironment(t *testing.T) (
	*gorm.DB,
	repository.EventRepository,
	repository.VenueRepository,
	repository.PricingTierRepository,
) {
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

	if err := db.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	if err := db.AutoMigrate(&model.Venue{}, &model.Event{}, &model.PricingTier{}); err != nil {
		t.Fatalf("failed to automigrate schema: %v", err)
	}

	t.Cleanup(func() {
		sqlDB.Close()
	})

	eventRepo := repository.NewEventRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	tierRepo := repository.NewPricingTierRepository(db)

	return db, eventRepo, venueRepo, tierRepo
}

// createSeedVenue helper creates a canonical test venue
func createSeedVenue(t *testing.T, venueRepo repository.VenueRepository) *model.Venue {
	t.Helper()
	venue := &model.Venue{
		ID:            uuid.New(),
		Name:          "Gelora Bung Karno Stadium",
		Address:       "Jl. Pintu Satu Senayan",
		City:          "Jakarta Pusat",
		TotalCapacity: 77000,
		LayoutConfig:  `{"type":"stadium"}`,
	}
	if err := venueRepo.SeedVenues(context.Background(), []model.Venue{*venue}); err != nil {
		t.Fatalf("failed to seed venue: %v", err)
	}
	return venue
}

// ---------------------------------------------------------------------------
// 1. Event Lifecycle State Machine Tests
// ---------------------------------------------------------------------------

func TestEventService_LifecycleStateTransitions(t *testing.T) {
	_, eventRepo, venueRepo, _ := setupTestEnvironment(t)
	cacheService := service.NewInMemoryCacheService()
	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	ctx := context.Background()

	venue := createSeedVenue(t, venueRepo)

	// Step 1: Create event -> Initial state must be DRAFT
	createReq := dto.CreateEventRequest{
		Title:       "Coldplay Live in Jakarta",
		Description: "Music of the Spheres",
		VenueID:     venue.ID,
		EventDate:   time.Now().Add(60 * 24 * time.Hour),
	}
	created, err := eventService.CreateEvent(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if created.Status != model.StatusDraft {
		t.Fatalf("expected initial status DRAFT, got: %s", created.Status)
	}

	eventID := created.ID

	// Step 2: Test illegal direct transitions from DRAFT
	illegalFromDraft := []model.EventStatus{
		model.StatusOnSale,
		model.StatusSoldOut,
		model.StatusPast,
	}
	for _, target := range illegalFromDraft {
		_, err := eventService.UpdateEventStatus(ctx, eventID, target)
		if !errors.Is(err, model.ErrInvalidStateTransition) {
			t.Errorf("expected ErrInvalidStateTransition from DRAFT to %s, got: %v", target, err)
		}
	}

	// Step 3: Transition DRAFT -> PUBLISHED (Positive path)
	pubResp, err := eventService.UpdateEventStatus(ctx, eventID, model.StatusPublished)
	if err != nil {
		t.Fatalf("transition to PUBLISHED failed: %v", err)
	}
	if pubResp.Status != model.StatusPublished {
		t.Fatalf("expected status PUBLISHED, got: %s", pubResp.Status)
	}

	// Step 4: Test illegal transitions from PUBLISHED
	illegalFromPublished := []model.EventStatus{
		model.StatusDraft,
		model.StatusSoldOut,
		model.StatusPast,
	}
	for _, target := range illegalFromPublished {
		_, err := eventService.UpdateEventStatus(ctx, eventID, target)
		if !errors.Is(err, model.ErrInvalidStateTransition) {
			t.Errorf("expected ErrInvalidStateTransition from PUBLISHED to %s, got: %v", target, err)
		}
	}

	// Step 5: Transition PUBLISHED -> ON_SALE (Positive path)
	onSaleResp, err := eventService.UpdateEventStatus(ctx, eventID, model.StatusOnSale)
	if err != nil {
		t.Fatalf("transition to ON_SALE failed: %v", err)
	}
	if onSaleResp.Status != model.StatusOnSale {
		t.Fatalf("expected status ON_SALE, got: %s", onSaleResp.Status)
	}

	// Step 6: Test illegal transitions from ON_SALE
	illegalFromOnSale := []model.EventStatus{
		model.StatusDraft,
		model.StatusPublished,
	}
	for _, target := range illegalFromOnSale {
		_, err := eventService.UpdateEventStatus(ctx, eventID, target)
		if !errors.Is(err, model.ErrInvalidStateTransition) {
			t.Errorf("expected ErrInvalidStateTransition from ON_SALE to %s, got: %v", target, err)
		}
	}

	// Step 7: Transition ON_SALE -> SOLD_OUT (Positive path)
	soldOutResp, err := eventService.UpdateEventStatus(ctx, eventID, model.StatusSoldOut)
	if err != nil {
		t.Fatalf("transition to SOLD_OUT failed: %v", err)
	}
	if soldOutResp.Status != model.StatusSoldOut {
		t.Fatalf("expected status SOLD_OUT, got: %s", soldOutResp.Status)
	}

	// Step 8: Reopen ticket sales: SOLD_OUT -> ON_SALE (Positive path)
	reopenedResp, err := eventService.UpdateEventStatus(ctx, eventID, model.StatusOnSale)
	if err != nil {
		t.Fatalf("reopening sales (SOLD_OUT -> ON_SALE) failed: %v", err)
	}
	if reopenedResp.Status != model.StatusOnSale {
		t.Fatalf("expected status ON_SALE after reopen, got: %s", reopenedResp.Status)
	}

	// Step 9: Transition ON_SALE -> PAST (Positive path: Event concluded)
	pastResp, err := eventService.UpdateEventStatus(ctx, eventID, model.StatusPast)
	if err != nil {
		t.Fatalf("transition to PAST failed: %v", err)
	}
	if pastResp.Status != model.StatusPast {
		t.Fatalf("expected status PAST, got: %s", pastResp.Status)
	}

	// Step 10: PAST is a terminal state -> No further transitions permitted
	allStatuses := []model.EventStatus{
		model.StatusDraft,
		model.StatusPublished,
		model.StatusOnSale,
		model.StatusSoldOut,
		model.StatusCancelled,
		model.StatusPast,
	}
	for _, target := range allStatuses {
		_, err := eventService.UpdateEventStatus(ctx, eventID, target)
		if !errors.Is(err, model.ErrTerminalState) {
			t.Errorf("expected ErrTerminalState from PAST to %s, got: %v", target, err)
		}
	}

	// Step 11: Test CANCELLED terminal state on a separate event
	cancelEvent, err := eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:     "Cancelled Tour",
		VenueID:   venue.ID,
		EventDate: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	// Direct DRAFT -> CANCELLED is allowed
	cancelledResp, err := eventService.UpdateEventStatus(ctx, cancelEvent.ID, model.StatusCancelled)
	if err != nil {
		t.Fatalf("transition to CANCELLED failed: %v", err)
	}
	if cancelledResp.Status != model.StatusCancelled {
		t.Fatalf("expected status CANCELLED, got: %s", cancelledResp.Status)
	}

	// Once CANCELLED, any modification returns ErrTerminalState
	for _, target := range allStatuses {
		_, err := eventService.UpdateEventStatus(ctx, cancelEvent.ID, target)
		if !errors.Is(err, model.ErrTerminalState) {
			t.Errorf("expected ErrTerminalState from CANCELLED to %s, got: %v", target, err)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. Pricing Tier Lock Rules & Validation Tests
// ---------------------------------------------------------------------------

func TestPricingTierService_LockRule_And_Validation(t *testing.T) {
	_, eventRepo, venueRepo, tierRepo := setupTestEnvironment(t)
	cacheService := service.NewInMemoryCacheService()
	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	tierService := service.NewPricingTierService(tierRepo, eventRepo, cacheService)
	ctx := context.Background()

	venue := createSeedVenue(t, venueRepo)

	// Create Event in DRAFT
	eventResp, err := eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:     "Maroon 5 World Tour",
		VenueID:   venue.ID,
		EventDate: time.Now().Add(45 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	eventID := eventResp.ID

	// 1. Validation error cases
	// A. Negative price
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "VIP",
		Price:      -100,
		TotalSeats: 100,
	})
	if !errors.Is(err, model.ErrInvalidTierPrice) {
		t.Errorf("expected ErrInvalidTierPrice for negative price, got: %v", err)
	}

	// B. Zero total seats
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "VIP",
		Price:      1000000,
		TotalSeats: 0,
	})
	if !errors.Is(err, model.ErrInvalidTierSeats) {
		t.Errorf("expected ErrInvalidTierSeats for zero seats, got: %v", err)
	}

	// C. Empty tier name
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "   ",
		Price:      500000,
		TotalSeats: 100,
	})
	if !errors.Is(err, model.ErrInvalidTierName) {
		t.Errorf("expected ErrInvalidTierName for empty name, got: %v", err)
	}

	// 2. Add tier while in DRAFT status -> MUST SUCCEED
	tier1, err := tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "Festival",
		Price:      750000,
		Currency:   "IDR",
		TotalSeats: 5000,
	})
	if err != nil {
		t.Fatalf("AddPricingTier in DRAFT status failed: %v", err)
	}
	if tier1.TierName != "Festival" || tier1.Price != 750000 {
		t.Fatalf("unexpected tier fields: %+v", tier1)
	}

	// 3. Transition to PUBLISHED status
	_, err = eventService.UpdateEventStatus(ctx, eventID, model.StatusPublished)
	if err != nil {
		t.Fatalf("transition to PUBLISHED failed: %v", err)
	}

	// Add tier while in PUBLISHED status -> MUST SUCCEED
	tier2, err := tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "VIP",
		Price:      1750000,
		Currency:   "IDR",
		TotalSeats: 1000,
	})
	if err != nil {
		t.Fatalf("AddPricingTier in PUBLISHED status failed: %v", err)
	}
	if tier2.TierName != "VIP" {
		t.Fatalf("expected VIP, got: %s", tier2.TierName)
	}

	// Verify tiers list returns 2 tiers
	tiers, err := tierService.GetPricingTiers(ctx, eventID)
	if err != nil {
		t.Fatalf("GetPricingTiers failed: %v", err)
	}
	if len(tiers) != 2 {
		t.Fatalf("expected 2 tiers, got: %d", len(tiers))
	}

	// 4. Transition to ON_SALE status
	_, err = eventService.UpdateEventStatus(ctx, eventID, model.StatusOnSale)
	if err != nil {
		t.Fatalf("transition to ON_SALE failed: %v", err)
	}

	// Adding pricing tier when ON_SALE MUST RETURN ErrPricingTierLocked
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "Late Bird",
		Price:      2000000,
		TotalSeats: 500,
	})
	if !errors.Is(err, model.ErrPricingTierLocked) {
		t.Fatalf("expected ErrPricingTierLocked for ON_SALE event, got: %v", err)
	}

	// 5. Transition to SOLD_OUT status
	_, err = eventService.UpdateEventStatus(ctx, eventID, model.StatusSoldOut)
	if err != nil {
		t.Fatalf("transition to SOLD_OUT failed: %v", err)
	}

	// Adding pricing tier when SOLD_OUT MUST RETURN ErrPricingTierLocked
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "Extra Seats",
		Price:      2500000,
		TotalSeats: 200,
	})
	if !errors.Is(err, model.ErrPricingTierLocked) {
		t.Fatalf("expected ErrPricingTierLocked for SOLD_OUT event, got: %v", err)
	}

	// 6. Test non-existent event ID
	_, err = tierService.AddPricingTier(ctx, uuid.New(), dto.CreatePricingTierRequest{
		TierName:   "Ghost Tier",
		Price:      100000,
		TotalSeats: 10,
	})
	if !errors.Is(err, model.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound for non-existent event, got: %v", err)
	}

	_, err = tierService.GetPricingTiers(ctx, uuid.New())
	if !errors.Is(err, model.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound for non-existent event query, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 3. Redis Cache Set, Hit, Miss, and Mutation-Triggered Invalidation Tests
// ---------------------------------------------------------------------------

func TestCache_HitMiss_And_MutationInvalidation(t *testing.T) {
	db, eventRepo, venueRepo, tierRepo := setupTestEnvironment(t)
	cacheService := service.NewInMemoryCacheService()
	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	tierService := service.NewPricingTierService(tierRepo, eventRepo, cacheService)
	ctx := context.Background()

	venue := createSeedVenue(t, venueRepo)

	// Create event
	created, err := eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:     "Java Jazz Festival 2026",
		VenueID:   venue.ID,
		EventDate: time.Now().Add(100 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	eventID := created.ID

	// Add initial tier
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "Regular",
		Price:      450000,
		TotalSeats: 3000,
	})
	if err != nil {
		t.Fatalf("AddPricingTier failed: %v", err)
	}

	// 1. First call to GetEventByID: Cache MISS -> Populates cache
	detail1, err := eventService.GetEventByID(ctx, eventID)
	if err != nil {
		t.Fatalf("first GetEventByID failed: %v", err)
	}
	if detail1.Title != "Java Jazz Festival 2026" {
		t.Fatalf("unexpected event title: %s", detail1.Title)
	}
	if len(detail1.PricingTiers) != 1 {
		t.Fatalf("expected 1 pricing tier, got: %d", len(detail1.PricingTiers))
	}

	// Verify key was set in cache
	var cachedDetail dto.EventDetailResponse
	cacheKey := service.EventKeyPrefix + eventID.String()
	if !cacheService.Get(ctx, cacheKey, &cachedDetail) {
		t.Fatal("expected event detail to be present in cache after GetEventByID")
	}

	// Mutate DB directly behind the cache's back to prove subsequent call is a Cache HIT
	if err := db.Model(&model.Event{}).Where("id = ?", eventID).Update("name", "DB Ghost Changed Name").Error; err != nil {
		t.Fatalf("direct DB update failed: %v", err)
	}

	// 2. Second call to GetEventByID: Cache HIT -> returns cached title, NOT DB changed title
	detail2, err := eventService.GetEventByID(ctx, eventID)
	if err != nil {
		t.Fatalf("second GetEventByID failed: %v", err)
	}
	if detail2.Title != "Java Jazz Festival 2026" {
		t.Fatalf("expected Cache HIT with original title, got changed DB name: %s", detail2.Title)
	}

	// 3. Mutate event status via service -> MUST trigger cache invalidation
	_, err = eventService.UpdateEventStatus(ctx, eventID, model.StatusPublished)
	if err != nil {
		t.Fatalf("UpdateEventStatus failed: %v", err)
	}

	// Verify that the cache key was evicted by UpdateEventStatus
	if cacheService.Get(ctx, cacheKey, &cachedDetail) {
		t.Fatal("expected cache key to be evicted after UpdateEventStatus")
	}

	// 4. Query again: Cache MISS -> Fetches fresh data from DB
	detail3, err := eventService.GetEventByID(ctx, eventID)
	if err != nil {
		t.Fatalf("GetEventByID post-invalidation failed: %v", err)
	}
	if detail3.Status != model.StatusPublished {
		t.Fatalf("expected fresh status PUBLISHED from DB, got: %s", detail3.Status)
	}

	// 5. Test Search Catalog caching and mutation invalidation
	searchFilter := dto.EventFilter{City: "jakarta"}
	// First search -> MISS and cached
	searchRes1, err := eventService.SearchEvents(ctx, searchFilter, 1, 10)
	if err != nil {
		t.Fatalf("SearchEvents failed: %v", err)
	}
	if len(searchRes1.Items) != 1 {
		t.Fatalf("expected 1 search item, got: %d", len(searchRes1.Items))
	}

	searchCacheKey := service.GenerateCatalogCacheKey(searchFilter, 1, 10)
	var cachedSearch dto.PaginatedEventsResponse
	if !cacheService.Get(ctx, searchCacheKey, &cachedSearch) {
		t.Fatal("expected search query to be cached")
	}

	// Adding a pricing tier should invalidate the catalog search query cache
	_, err = tierService.AddPricingTier(ctx, eventID, dto.CreatePricingTierRequest{
		TierName:   "VIP Premium",
		Price:      1200000,
		TotalSeats: 500,
	})
	if err != nil {
		t.Fatalf("AddPricingTier failed: %v", err)
	}

	// Verify that the catalog search cache key was purged
	if cacheService.Get(ctx, searchCacheKey, &cachedSearch) {
		t.Fatal("expected search query cache to be purged following AddPricingTier mutation")
	}
}

// ---------------------------------------------------------------------------
// 4. Graceful Fallback When Redis Is Offline Tests
// ---------------------------------------------------------------------------

func TestCache_GracefulDegradation_RedisOffline(t *testing.T) {
	_, eventRepo, venueRepo, tierRepo := setupTestEnvironment(t)
	venue := createSeedVenue(t, venueRepo)
	ctx := context.Background()

	// Scenario A: Redis client is nil
	t.Run("NilRedisClient_Fallback", func(t *testing.T) {
		nilCacheService := service.NewRedisCacheService(nil)

		// 1. Direct cache operations on nil client must not panic or error
		var dest dto.EventDetailResponse
		if nilCacheService.Get(ctx, "any:key", &dest) {
			t.Fatal("expected Get to return false on nil client")
		}
		if err := nilCacheService.Set(ctx, "any:key", "value", 60*time.Second); err != nil {
			t.Fatalf("expected Set to return nil on nil client: %v", err)
		}
		if err := nilCacheService.InvalidateEvent(ctx, uuid.New()); err != nil {
			t.Fatalf("expected InvalidateEvent to return nil on nil client: %v", err)
		}
		if err := nilCacheService.InvalidateCatalog(ctx); err != nil {
			t.Fatalf("expected InvalidateCatalog to return nil on nil client: %v", err)
		}

		// 2. EventService and PricingTierService work normally without cache
		eventSvc := service.NewEventService(eventRepo, venueRepo, nilCacheService)
		tierSvc := service.NewPricingTierService(tierRepo, eventRepo, nilCacheService)

		created, err := eventSvc.CreateEvent(ctx, dto.CreateEventRequest{
			Title:     "Standalone Acoustic Night",
			VenueID:   venue.ID,
			EventDate: time.Now().Add(10 * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateEvent on nil cache failed: %v", err)
		}

		detail, err := eventSvc.GetEventByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetEventByID on nil cache failed: %v", err)
		}
		if detail.Title != "Standalone Acoustic Night" {
			t.Fatalf("expected correct title from DB, got: %s", detail.Title)
		}

		// Tier operations on nil cache
		tier, err := tierSvc.AddPricingTier(ctx, created.ID, dto.CreatePricingTierRequest{
			TierName:   "General Admission",
			Price:      250000,
			TotalSeats: 500,
		})
		if err != nil {
			t.Fatalf("AddPricingTier on nil cache failed: %v", err)
		}
		if tier.TierName != "General Admission" {
			t.Fatalf("unexpected tier name: %s", tier.TierName)
		}

		tiers, err := tierSvc.GetPricingTiers(ctx, created.ID)
		if err != nil || len(tiers) != 1 {
			t.Fatalf("GetPricingTiers on nil cache failed: len=%d, err=%v", len(tiers), err)
		}
	})

	// Scenario B: Redis client configured with unreachable / offline address
	t.Run("UnreachableRedisClient_Fallback", func(t *testing.T) {
		// Use pkgredis.NewClient pointing to an unreachable port (e.g. 127.0.0.1:6398)
		offlineClient := pkgredis.NewClient("127.0.0.1:6398")
		defer offlineClient.Close()

		offlineCacheService := service.NewRedisCacheService(offlineClient)

		// 1. Direct operations must gracefully fail without panics or bubbles
		var dest dto.EventResponse
		hit := offlineCacheService.Get(ctx, "catalog:event:offline", &dest)
		if hit {
			t.Fatal("expected Get on offline Redis to return false (cache miss)")
		}

		err := offlineCacheService.Set(ctx, "catalog:event:offline", "test-val", 10*time.Second)
		if err != nil {
			t.Fatalf("expected Set on offline Redis to transparently return nil, got: %v", err)
		}

		// 2. Services transparently fallback to PostgreSQL DB
		eventSvc := service.NewEventService(eventRepo, venueRepo, offlineCacheService)
		searchRes, err := eventSvc.SearchEvents(ctx, dto.EventFilter{}, 1, 10)
		if err != nil {
			t.Fatalf("SearchEvents failed with offline Redis: %v", err)
		}
		if len(searchRes.Items) == 0 {
			t.Fatal("expected DB search results even when Redis is offline")
		}
	})
}

// ---------------------------------------------------------------------------
// 5. Venue and Event Validation Edge Cases
// ---------------------------------------------------------------------------

func TestEventService_ValidationEdgeCases(t *testing.T) {
	_, eventRepo, venueRepo, _ := setupTestEnvironment(t)
	cacheService := service.NewInMemoryCacheService()
	eventService := service.NewEventService(eventRepo, venueRepo, cacheService)
	ctx := context.Background()

	venue := createSeedVenue(t, venueRepo)

	// 1. Missing Title
	_, err := eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:     "   ",
		VenueID:   venue.ID,
		EventDate: time.Now().Add(24 * time.Hour),
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}

	// 2. Nil VenueID
	_, err = eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:     "Concert",
		VenueID:   uuid.Nil,
		EventDate: time.Now().Add(24 * time.Hour),
	})
	if err == nil {
		t.Fatal("expected error for nil venue_id")
	}

	// 3. Non-existent VenueID
	_, err = eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:     "Concert",
		VenueID:   uuid.New(),
		EventDate: time.Now().Add(24 * time.Hour),
	})
	if !errors.Is(err, service.ErrVenueNotFound) {
		t.Fatalf("expected ErrVenueNotFound, got: %v", err)
	}

	// 4. Zero EventDate
	_, err = eventService.CreateEvent(ctx, dto.CreateEventRequest{
		Title:   "Concert",
		VenueID: venue.ID,
	})
	if err == nil {
		t.Fatal("expected error for zero event_date")
	}

	// 5. Get non-existent event
	_, err = eventService.GetEventByID(ctx, uuid.New())
	if !errors.Is(err, model.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound for non-existent event, got: %v", err)
	}

	// 6. UpdateStatus on non-existent event
	_, err = eventService.UpdateEventStatus(ctx, uuid.New(), model.StatusPublished)
	if !errors.Is(err, model.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound on non-existent update status, got: %v", err)
	}

	// 7. SearchEvents bounds checking
	paginated, err := eventService.SearchEvents(ctx, dto.EventFilter{}, -5, 500)
	if err != nil {
		t.Fatalf("SearchEvents bounds failed: %v", err)
	}
	if paginated.Meta.CurrentPage != 1 {
		t.Errorf("expected page clamped to 1, got: %d", paginated.Meta.CurrentPage)
	}
	if paginated.Meta.Limit != 100 {
		t.Errorf("expected limit clamped to 100, got: %d", paginated.Meta.Limit)
	}
}
