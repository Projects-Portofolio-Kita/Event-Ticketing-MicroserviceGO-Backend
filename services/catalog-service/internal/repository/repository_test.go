package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
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

	return db
}

// ---------------------------------------------------------------------------
// VenueRepository Tests
// ---------------------------------------------------------------------------

func TestVenueRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewVenueRepository(db)
	ctx := context.Background()

	// 1. Initial count should be 0
	cnt, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("expected 0 venues, got %d", cnt)
	}

	// 2. Seed empty slice should be a no-op
	if err := repo.SeedVenues(ctx, []model.Venue{}); err != nil {
		t.Fatalf("seeding empty slice should succeed: %v", err)
	}

	// 3. Seed venues
	venue1ID := uuid.New()
	venue2ID := uuid.New()
	venues := []model.Venue{
		{
			ID:            venue1ID,
			Name:          "Gelora Bung Karno",
			Address:       "Senayan",
			City:          "Jakarta Pusat",
			TotalCapacity: 77000,
			LayoutConfig:  `{"type":"stadium"}`,
		},
		{
			ID:            venue2ID,
			Name:          "ICE BSD",
			Address:       "BSD City",
			City:          "Tangerang",
			TotalCapacity: 10000,
			LayoutConfig:  `{"type":"hall"}`,
		},
	}

	if err := repo.SeedVenues(ctx, venues); err != nil {
		t.Fatalf("SeedVenues failed: %v", err)
	}

	// Verify count
	cnt, err = repo.Count(ctx)
	if err != nil || cnt != 2 {
		t.Fatalf("expected count 2, got %d (err: %v)", cnt, err)
	}

	// 4. List venues (ordered by name ASC: Gelora Bung Karno, ICE BSD)
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 venues in list, got %d", len(list))
	}
	if list[0].Name != "Gelora Bung Karno" || list[1].Name != "ICE BSD" {
		t.Fatalf("unexpected order: %s, %s", list[0].Name, list[1].Name)
	}

	// 5. GetByID existing
	v1, err := repo.GetByID(ctx, venue1ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if v1.Name != "Gelora Bung Karno" {
		t.Fatalf("expected 'Gelora Bung Karno', got %s", v1.Name)
	}

	// 6. GetByID non-existing
	_, err = repo.GetByID(ctx, uuid.New())
	if err != repository.ErrVenueNotFound {
		t.Fatalf("expected ErrVenueNotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PricingTierRepository Tests
// ---------------------------------------------------------------------------

func TestPricingTierRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	venueRepo := repository.NewVenueRepository(db)
	eventRepo := repository.NewEventRepository(db)
	tierRepo := repository.NewPricingTierRepository(db)
	ctx := context.Background()

	// Prepare venue & event
	venue := model.Venue{
		ID:            uuid.New(),
		Name:          "JIExpo",
		Address:       "Kemayoran",
		City:          "Jakarta Pusat",
		TotalCapacity: 12000,
	}
	if err := venueRepo.SeedVenues(ctx, []model.Venue{venue}); err != nil {
		t.Fatalf("failed to seed venue: %v", err)
	}

	event := model.Event{
		ID:          uuid.New(),
		Name:        "Java Jazz Festival",
		Description: "Annual jazz festival",
		VenueID:     venue.ID,
		Status:      model.StatusDraft,
		EventDate:   time.Now().Add(48 * time.Hour),
	}
	if err := eventRepo.Create(ctx, &event); err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	// 1. Create nil tier
	if err := tierRepo.Create(ctx, nil); err != model.ErrNilPricingTier {
		t.Fatalf("expected ErrNilPricingTier, got: %v", err)
	}

	// 2. Create invalid tier
	invalidTier := &model.PricingTier{
		EventID:    event.ID,
		TierName:   "   ", // whitespace only
		Price:      100000,
		TotalSeats: 50,
	}
	if err := tierRepo.Create(ctx, invalidTier); err != model.ErrInvalidTierName {
		t.Fatalf("expected ErrInvalidTierName, got: %v", err)
	}

	// 3. Create valid tiers
	tierVIP := &model.PricingTier{
		EventID:    event.ID,
		TierName:   "VIP",
		Price:      1500000,
		Currency:   "IDR",
		TotalSeats: 100,
	}
	if err := tierRepo.Create(ctx, tierVIP); err != nil {
		t.Fatalf("failed to create tierVIP: %v", err)
	}
	if tierVIP.ID == uuid.Nil {
		t.Fatal("expected UUID to be assigned")
	}

	tierReg := &model.PricingTier{
		EventID:    event.ID,
		TierName:   "Regular",
		Price:      500000,
		Currency:   "IDR",
		TotalSeats: 500,
	}
	if err := tierRepo.Create(ctx, tierReg); err != nil {
		t.Fatalf("failed to create tierReg: %v", err)
	}

	// 4. GetByID
	fetched, err := tierRepo.GetByID(ctx, tierVIP.ID)
	if err != nil {
		t.Fatalf("failed to GetByID: %v", err)
	}
	if fetched.TierName != "VIP" || fetched.Price != 1500000 {
		t.Fatalf("mismatched tier: %+v", fetched)
	}

	// 5. GetByID not found
	_, err = tierRepo.GetByID(ctx, uuid.New())
	if err != repository.ErrPricingTierNotFound {
		t.Fatalf("expected ErrPricingTierNotFound, got: %v", err)
	}

	// 6. GetByEventID (ordered by price ASC: Regular 500000, VIP 1500000)
	tiers, err := tierRepo.GetByEventID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByEventID failed: %v", err)
	}
	if len(tiers) != 2 {
		t.Fatalf("expected 2 tiers, got %d", len(tiers))
	}
	if tiers[0].TierName != "Regular" || tiers[1].TierName != "VIP" {
		t.Fatalf("expected Regular then VIP, got %s then %s", tiers[0].TierName, tiers[1].TierName)
	}

	// 7. Update tier - verify zero-value preservation (e.g. Free Tier price = 0)
	tierReg.Price = 0
	tierReg.TierName = "Regular Free Promo"
	if err := tierRepo.Update(ctx, tierReg); err != nil {
		t.Fatalf("failed to update tier: %v", err)
	}

	updated, err := tierRepo.GetByID(ctx, tierReg.ID)
	if err != nil {
		t.Fatalf("failed to get updated tier: %v", err)
	}
	if updated.Price != 0 {
		t.Fatalf("expected price to be updated to 0.00, got: %f", updated.Price)
	}
	if updated.TierName != "Regular Free Promo" {
		t.Fatalf("expected name to be updated, got: %s", updated.TierName)
	}

	// Update nil tier
	if err := tierRepo.Update(ctx, nil); err != model.ErrNilPricingTier {
		t.Fatalf("expected ErrNilPricingTier on Update(nil), got: %v", err)
	}

	// Update non-existent tier
	nonExistentTier := &model.PricingTier{
		ID:         uuid.New(),
		EventID:    event.ID,
		TierName:   "Ghost",
		Price:      100,
		TotalSeats: 10,
	}
	if err := tierRepo.Update(ctx, nonExistentTier); err != repository.ErrPricingTierNotFound {
		t.Fatalf("expected ErrPricingTierNotFound, got: %v", err)
	}

	// 8. Delete tier
	if err := tierRepo.Delete(ctx, tierReg.ID); err != nil {
		t.Fatalf("failed to delete tier: %v", err)
	}

	// Verify it's gone
	_, err = tierRepo.GetByID(ctx, tierReg.ID)
	if err != repository.ErrPricingTierNotFound {
		t.Fatalf("expected ErrPricingTierNotFound after delete, got: %v", err)
	}

	// Delete non-existent
	if err := tierRepo.Delete(ctx, uuid.New()); err != repository.ErrPricingTierNotFound {
		t.Fatalf("expected ErrPricingTierNotFound deleting non-existent, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EventRepository Tests
// ---------------------------------------------------------------------------

func TestEventRepository_CRUD_And_Preloads(t *testing.T) {
	db := setupTestDB(t)
	venueRepo := repository.NewVenueRepository(db)
	eventRepo := repository.NewEventRepository(db)
	tierRepo := repository.NewPricingTierRepository(db)
	ctx := context.Background()

	// 1. Create with nil event
	if err := eventRepo.Create(ctx, nil); err != model.ErrNilEvent {
		t.Fatalf("expected ErrNilEvent on Create(nil), got: %v", err)
	}

	// Prepare venue
	venue := model.Venue{
		ID:            uuid.New(),
		Name:          "Gelora Bung Karno",
		Address:       "Senayan",
		City:          "Jakarta Pusat",
		TotalCapacity: 77000,
	}
	if err := venueRepo.SeedVenues(ctx, []model.Venue{venue}); err != nil {
		t.Fatalf("SeedVenues failed: %v", err)
	}

	// 2. Create Event
	now := time.Now().Truncate(time.Second)
	event := model.Event{
		Name:        "Coldplay Live 2026",
		Description: "Music of the Spheres World Tour",
		VenueID:     venue.ID,
		EventDate:   now.Add(30 * 24 * time.Hour),
		StartDate:   now.Add(30*24*time.Hour + 19*time.Hour),
		EndDate:     now.Add(30*24*time.Hour + 23*time.Hour),
	}
	if err := eventRepo.Create(ctx, &event); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if event.ID == uuid.Nil {
		t.Fatal("expected UUID to be assigned")
	}
	if event.Status != model.StatusDraft {
		t.Fatalf("expected default status DRAFT, got: %s", event.Status)
	}

	// Add Pricing Tiers
	t1 := &model.PricingTier{
		EventID:    event.ID,
		TierName:   "VIP",
		Price:      2500000,
		TotalSeats: 200,
	}
	t2 := &model.PricingTier{
		EventID:    event.ID,
		TierName:   "Festival",
		Price:      1000000,
		TotalSeats: 1000,
	}
	if err := tierRepo.Create(ctx, t1); err != nil {
		t.Fatalf("failed to create t1: %v", err)
	}
	if err := tierRepo.Create(ctx, t2); err != nil {
		t.Fatalf("failed to create t2: %v", err)
	}

	// 3. GetByID with preloads
	fetched, err := eventRepo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Name != "Coldplay Live 2026" {
		t.Fatalf("expected name Coldplay Live 2026, got %s", fetched.Name)
	}
	if fetched.Venue.ID != venue.ID || fetched.Venue.Name != "Gelora Bung Karno" {
		t.Fatalf("Venue association not preloaded properly: %+v", fetched.Venue)
	}
	if len(fetched.PricingTiers) != 2 {
		t.Fatalf("expected 2 preloaded pricing tiers, got %d", len(fetched.PricingTiers))
	}

	// 4. GetByID not found
	_, err = eventRepo.GetByID(ctx, uuid.New())
	if err != repository.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got: %v", err)
	}

	// 5. Update
	fetched.Name = "Coldplay Live in Jakarta - Day 1"
	fetched.Description = "Updated description"
	if err := eventRepo.Update(ctx, fetched); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	refetched, err := eventRepo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if refetched.Name != "Coldplay Live in Jakarta - Day 1" || refetched.Description != "Updated description" {
		t.Fatalf("mismatched updated fields: %+v", refetched)
	}

	// Update nil event
	if err := eventRepo.Update(ctx, nil); err != model.ErrNilEvent {
		t.Fatalf("expected ErrNilEvent on Update(nil), got: %v", err)
	}

	// Update non-existent event
	nonExistentEvent := &model.Event{
		ID:      uuid.New(),
		Name:    "Ghost Event",
		VenueID: venue.ID,
	}
	if err := eventRepo.Update(ctx, nonExistentEvent); err != repository.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound on non-existent update, got: %v", err)
	}

	// 6. UpdateStatus
	if err := eventRepo.UpdateStatus(ctx, event.ID, model.StatusPublished); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	refetchedStatus, err := eventRepo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if refetchedStatus.Status != model.StatusPublished {
		t.Fatalf("expected status PUBLISHED, got: %s", refetchedStatus.Status)
	}

	// UpdateStatus non-existent
	if err := eventRepo.UpdateStatus(ctx, uuid.New(), model.StatusPublished); err != repository.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound on UpdateStatus non-existent, got: %v", err)
	}
}

func TestEventRepository_List_FilteringAndPagination(t *testing.T) {
	db := setupTestDB(t)
	venueRepo := repository.NewVenueRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ctx := context.Background()

	// Seed Venues in different cities
	venueJkt := model.Venue{
		ID:            uuid.New(),
		Name:          "GBK Arena",
		Address:       "Senayan",
		City:          "Jakarta Pusat",
		TotalCapacity: 50000,
	}
	venueBali := model.Venue{
		ID:            uuid.New(),
		Name:          "GWK Cultural Park",
		Address:       "Uluwatu",
		City:          "Badung Bali",
		TotalCapacity: 25000,
	}
	if err := venueRepo.SeedVenues(ctx, []model.Venue{venueJkt, venueBali}); err != nil {
		t.Fatalf("failed to seed venues: %v", err)
	}

	baseDate := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	// Create 5 test events
	events := []model.Event{
		{
			Name:      "Jakarta Jazz Night",
			VenueID:   venueJkt.ID,
			Status:    model.StatusDraft,
			EventDate: baseDate.AddDate(0, 0, 1), // 2026-11-02
		},
		{
			Name:      "Jakarta Rock Fest",
			VenueID:   venueJkt.ID,
			Status:    model.StatusPublished,
			EventDate: baseDate.AddDate(0, 0, 5), // 2026-11-06
		},
		{
			Name:      "Jakarta EDM Rave",
			VenueID:   venueJkt.ID,
			Status:    model.StatusOnSale,
			EventDate: baseDate.AddDate(0, 0, 10), // 2026-11-11
		},
		{
			Name:      "Bali Sunset Chill",
			VenueID:   venueBali.ID,
			Status:    model.StatusPublished,
			EventDate: baseDate.AddDate(0, 0, 8), // 2026-11-09
		},
		{
			Name:      "Bali Dance Party",
			VenueID:   venueBali.ID,
			Status:    model.StatusOnSale,
			EventDate: baseDate.AddDate(0, 0, 20), // 2026-11-21
		},
	}

	for i := range events {
		if err := eventRepo.Create(ctx, &events[i]); err != nil {
			t.Fatalf("failed to create event %d: %v", i, err)
		}
	}

	// 1. List with empty filter -> all 5 events
	allEvents, total, err := eventRepo.List(ctx, repository.EventFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if total != 5 || len(allEvents) != 5 {
		t.Fatalf("expected total 5, len 5, got total=%d len=%d", total, len(allEvents))
	}
	// Verify Venue association is preloaded
	if allEvents[0].Venue.Name == "" {
		t.Fatal("expected Venue to be preloaded in list results")
	}

	// 2. Filter by Status (PUBLISHED -> 2 events)
	pubEvents, totalPub, err := eventRepo.List(ctx, repository.EventFilter{Status: model.StatusPublished}, 1, 10)
	if err != nil {
		t.Fatalf("List by status failed: %v", err)
	}
	if totalPub != 2 || len(pubEvents) != 2 {
		t.Fatalf("expected 2 published events, got total=%d len=%d", totalPub, len(pubEvents))
	}

	// 3. Filter by City (case-insensitive search: "jakarta")
	jktEvents, totalJkt, err := eventRepo.List(ctx, repository.EventFilter{City: "jAkArTa"}, 1, 10)
	if err != nil {
		t.Fatalf("List by city failed: %v", err)
	}
	if totalJkt != 3 || len(jktEvents) != 3 {
		t.Fatalf("expected 3 Jakarta events, got total=%d len=%d", totalJkt, len(jktEvents))
	}

	// 4. Filter by City: "bali"
	baliEvents, totalBali, err := eventRepo.List(ctx, repository.EventFilter{City: "BALI"}, 1, 10)
	if err != nil {
		t.Fatalf("List by city failed: %v", err)
	}
	if totalBali != 2 || len(baliEvents) != 2 {
		t.Fatalf("expected 2 Bali events, got total=%d len=%d", totalBali, len(baliEvents))
	}

	// 5. Filter by Date range (2026-11-05 to 2026-11-12)
	// Should match: Jakarta Rock Fest (11-06), Bali Sunset Chill (11-09), Jakarta EDM Rave (11-11) -> 3 events
	startDate := baseDate.AddDate(0, 0, 4)  // 2026-11-05
	endDate := baseDate.AddDate(0, 0, 12)    // 2026-11-13
	dateEvents, totalDate, err := eventRepo.List(ctx, repository.EventFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}, 1, 10)
	if err != nil {
		t.Fatalf("List by date range failed: %v", err)
	}
	if totalDate != 3 || len(dateEvents) != 3 {
		t.Fatalf("expected 3 events in date range, got total=%d len=%d", totalDate, len(dateEvents))
	}

	// 6. Filter by Search keyword
	searchEvents, totalSearch, err := eventRepo.List(ctx, repository.EventFilter{Search: "rock"}, 1, 10)
	if err != nil {
		t.Fatalf("List by search failed: %v", err)
	}
	if totalSearch != 1 || len(searchEvents) != 1 || searchEvents[0].Name != "Jakarta Rock Fest" {
		t.Fatalf("expected 1 rock event, got total=%d", totalSearch)
	}

	// 7. Combined filter: Status=ON_SALE and City=Jakarta
	combEvents, totalComb, err := eventRepo.List(ctx, repository.EventFilter{
		Status: model.StatusOnSale,
		City:   "jakarta",
	}, 1, 10)
	if err != nil {
		t.Fatalf("List combined failed: %v", err)
	}
	if totalComb != 1 || len(combEvents) != 1 || combEvents[0].Name != "Jakarta EDM Rave" {
		t.Fatalf("expected 1 combined match (Jakarta EDM Rave), got total=%d", totalComb)
	}

	// 8. Pagination: Page 1 Limit 2, Page 2 Limit 2, Page 3 Limit 2
	p1, t1, err := eventRepo.List(ctx, repository.EventFilter{}, 1, 2)
	if err != nil || t1 != 5 || len(p1) != 2 {
		t.Fatalf("Page 1 failed: total=%d, len=%d", t1, len(p1))
	}
	p2, t2, err := eventRepo.List(ctx, repository.EventFilter{}, 2, 2)
	if err != nil || t2 != 5 || len(p2) != 2 {
		t.Fatalf("Page 2 failed: total=%d, len=%d", t2, len(p2))
	}
	p3, t3, err := eventRepo.List(ctx, repository.EventFilter{}, 3, 2)
	if err != nil || t3 != 5 || len(p3) != 1 {
		t.Fatalf("Page 3 failed: total=%d, len=%d", t3, len(p3))
	}

	// Ensure no duplicate IDs between Page 1 and Page 2
	if p1[0].ID == p2[0].ID || p1[1].ID == p2[1].ID {
		t.Fatalf("pagination collision detected between page 1 and page 2")
	}
}
