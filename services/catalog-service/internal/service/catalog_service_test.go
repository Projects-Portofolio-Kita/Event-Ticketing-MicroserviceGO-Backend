package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakeCatalogStore is an in-memory eventStore; it mirrors the repository's
// error contract (gorm.ErrRecordNotFound for unknown rows).
type fakeCatalogStore struct {
	events        []model.Event
	eventsErr     error
	known         map[uuid.UUID]*model.Event
	lookupErr     error // non-not-found failure to propagate
	categories    map[uuid.UUID][]model.SeatCategory
	categoriesErr error
	seats         map[uuid.UUID][]model.Seat
	seatsErr      error
	createErr     error // failure to propagate from CreateEventWithSeatmap
	created       *struct {
		event      model.Event
		categories []model.SeatCategory
		seats      []model.Seat
	}
}

func (f *fakeCatalogStore) FindEvents(filter dto.EventFilter) ([]model.Event, error) {
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
	// The fake ignores paging but honours the text/city/category filters so
	// handler-level filtering tests can run without a database.
	events := make([]model.Event, 0, len(f.events))
	for _, event := range f.events {
		if filter.Q != "" && !strings.Contains(strings.ToLower(event.Name), strings.ToLower(filter.Q)) {
			continue
		}
		if filter.City != "" && !strings.EqualFold(event.City, filter.City) {
			continue
		}
		if filter.Category != "" && !strings.EqualFold(event.Category, filter.Category) {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func (f *fakeCatalogStore) FindEventCount(filter dto.EventFilter) (int64, error) {
	if f.eventsErr != nil {
		return 0, f.eventsErr
	}
	events, err := f.FindEvents(filter)
	if err != nil {
		return 0, err
	}
	return int64(len(events)), nil
}

func (f *fakeCatalogStore) FindDistinctCities() ([]dto.CityEntry, error) {
	counts := map[string]int{}
	order := []string{}
	for _, event := range f.events {
		if counts[event.City] == 0 {
			order = append(order, event.City)
		}
		counts[event.City]++
	}
	entries := make([]dto.CityEntry, 0, len(order))
	for _, city := range order {
		entries = append(entries, dto.CityEntry{City: city, Count: counts[city]})
	}
	return entries, nil
}

func (f *fakeCatalogStore) FindCategoriesByEvents(eventIDs []uuid.UUID) ([]model.SeatCategory, error) {
	wanted := map[uuid.UUID]bool{}
	for _, id := range eventIDs {
		wanted[id] = true
	}
	result := []model.SeatCategory{}
	for eventID, categories := range f.categories {
		if wanted[eventID] {
			result = append(result, categories...)
		}
	}
	return result, nil
}

func (f *fakeCatalogStore) FindByID(id uuid.UUID) (*model.Event, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	event, ok := f.known[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	stored := *event
	return &stored, nil
}

func (f *fakeCatalogStore) FindCategoriesByEvent(eventID uuid.UUID) ([]model.SeatCategory, error) {
	if f.categoriesErr != nil {
		return nil, f.categoriesErr
	}
	return f.categories[eventID], nil
}

func (f *fakeCatalogStore) FindSeatsByEvent(eventID uuid.UUID) ([]model.Seat, error) {
	if f.seatsErr != nil {
		return nil, f.seatsErr
	}
	return f.seats[eventID], nil
}

func (f *fakeCatalogStore) CodeExists(code string) (bool, error) {
	for _, event := range f.events {
		if strings.EqualFold(event.Code, code) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeCatalogStore) CreateEventWithSeatmap(event *model.Event, categories []model.SeatCategory, seats []model.Seat) error {
	if f.createErr != nil {
		return f.createErr
	}
	stored := *event
	f.events = append(f.events, stored)
	if f.known == nil {
		f.known = map[uuid.UUID]*model.Event{}
	}
	if f.categories == nil {
		f.categories = map[uuid.UUID][]model.SeatCategory{}
	}
	if f.seats == nil {
		f.seats = map[uuid.UUID][]model.Seat{}
	}
	f.known[event.ID] = &stored
	f.categories[event.ID] = categories
	f.seats[event.ID] = seats
	f.created = &struct {
		event      model.Event
		categories []model.SeatCategory
		seats      []model.Seat
	}{event: stored, categories: categories, seats: seats}
	return nil
}

// fakeBookingClient stands in for the booking-service availability feed.
type fakeBookingClient struct {
	statuses []client.SeatStatus
	tickets  []client.TicketStatus
	err      error
}

func (f *fakeBookingClient) GetSeatStatus(eventID string) ([]client.SeatStatus, error) {
	return f.statuses, f.err
}

func (f *fakeBookingClient) GetTicketStatus(eventID string) ([]client.TicketStatus, error) {
	return f.tickets, f.err
}

// seedStore builds one event with a single category holding two seats.
func seedStore(eventID uuid.UUID) *fakeCatalogStore {
	categoryID := uuid.New()
	return &fakeCatalogStore{
		events: []model.Event{{ID: eventID, Code: "COLDPLAY", Name: "Coldplay", Venue: "GBK"}},
		known:  map[uuid.UUID]*model.Event{eventID: {ID: eventID, Code: "COLDPLAY", Name: "Coldplay", Venue: "GBK"}},
		categories: map[uuid.UUID][]model.SeatCategory{
			eventID: {{ID: categoryID, EventID: eventID, Code: "CAT-A", Name: "VIP", Price: 100}},
		},
		seats: map[uuid.UUID][]model.Seat{
			eventID: {
				{ID: uuid.New(), EventID: eventID, CategoryID: categoryID, Label: "A1", RowName: "A", Number: 1, Side: "left"},
				{ID: uuid.New(), EventID: eventID, CategoryID: categoryID, Label: "A2", RowName: "A", Number: 2, Side: "right"},
			},
		},
	}
}

func newTestService(store *fakeCatalogStore, booking client.BookingClient) *CatalogService {
	if booking == nil {
		booking = &fakeBookingClient{}
	}
	return NewCatalogService(store, booking)
}

func TestListEventsMapsFields(t *testing.T) {
	store := seedStore(uuid.New())
	svc := newTestService(store, nil)

	events, total, err := svc.ListEvents(dto.EventFilter{})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if events[0].ID != store.events[0].ID.String() {
		t.Errorf("ID = %q, want %q", events[0].ID, store.events[0].ID)
	}
	if events[0].Code != "COLDPLAY" || events[0].Name != "Coldplay" || events[0].Venue != "GBK" {
		t.Errorf("event = %+v, want Coldplay at GBK", events[0])
	}
}

func TestListEventsEmptyCatalog(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{}, nil)

	events, total, err := svc.ListEvents(dto.EventFilter{})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if events == nil {
		t.Fatal("events = nil, want an empty non-nil slice")
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestListEventsPropagatesStoreError(t *testing.T) {
	boom := errors.New("db down")
	svc := newTestService(&fakeCatalogStore{eventsErr: boom}, nil)

	if _, _, err := svc.ListEvents(dto.EventFilter{}); !errors.Is(err, boom) {
		t.Fatalf("ListEvents = %v, want the store error", err)
	}
}

func TestGetSeatMapRejectsInvalidEventID(t *testing.T) {
	svc := newTestService(seedStore(uuid.New()), nil)

	for _, input := range []string{"", "not-a-uuid", "1234"} {
		categories, err := svc.GetSeatMap(input)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("GetSeatMap(%q) = %v, want ErrInvalidInput", input, err)
		}
		if categories != nil {
			t.Errorf("GetSeatMap(%q) categories = %v, want nil", input, categories)
		}
	}
}

func TestGetSeatMapUnknownEvent(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{known: map[uuid.UUID]*model.Event{}}, nil)

	categories, err := svc.GetSeatMap(uuid.New().String())
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("GetSeatMap = %v, want ErrEventNotFound", err)
	}
	if categories != nil {
		t.Errorf("categories = %v, want nil", categories)
	}
}

func TestGetSeatMapPropagatesStoreErrors(t *testing.T) {
	boom := errors.New("db down")

	t.Run("event lookup failure", func(t *testing.T) {
		svc := newTestService(&fakeCatalogStore{known: map[uuid.UUID]*model.Event{}, lookupErr: boom}, nil)
		if _, err := svc.GetSeatMap(uuid.New().String()); !errors.Is(err, boom) {
			t.Fatalf("GetSeatMap = %v, want the store error", err)
		}
	})

	t.Run("seat lookup failure", func(t *testing.T) {
		store := seedStore(uuid.New())
		store.seatsErr = boom
		svc := newTestService(store, nil)
		if _, err := svc.GetSeatMap(store.events[0].ID.String()); !errors.Is(err, boom) {
			t.Fatalf("GetSeatMap = %v, want the store error", err)
		}
	})
}

func TestGetSeatMapBuildsRows(t *testing.T) {
	store := seedStore(uuid.New())
	svc := newTestService(store, &fakeBookingClient{})

	categories, err := svc.GetSeatMap(store.events[0].ID.String())
	if err != nil {
		t.Fatalf("GetSeatMap: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("categories = %d, want 1", len(categories))
	}

	category := categories[0]
	if category.ID != "CAT-A" {
		t.Errorf("category ID = %q, want the category code CAT-A", category.ID)
	}
	if category.Name != "VIP" || category.Price != 100 {
		t.Errorf("category = %+v, want VIP at 100", category)
	}
	if len(category.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(category.Rows))
	}

	row := category.Rows[0]
	if row.RowName != "A" {
		t.Errorf("rowName = %q, want A", row.RowName)
	}
	if len(row.LeftSeats) != 1 || len(row.RightSeats) != 1 {
		t.Fatalf("left/right seats = %d/%d, want 1/1", len(row.LeftSeats), len(row.RightSeats))
	}
	if row.LeftSeats[0].ID != "A1" || row.RightSeats[0].ID != "A2" {
		t.Errorf("seats = %v / %v, want A1 / A2", row.LeftSeats[0].ID, row.RightSeats[0].ID)
	}
	if row.LeftSeats[0].Category != "VIP" {
		t.Errorf("seat category = %q, want VIP", row.LeftSeats[0].Category)
	}
	if row.LeftSeats[0].Status != "available" {
		t.Errorf("status = %q, want available", row.LeftSeats[0].Status)
	}
}

func TestGetSeatMapDegradesWhenBookingClientFails(t *testing.T) {
	store := seedStore(uuid.New())
	svc := newTestService(store, &fakeBookingClient{err: errors.New("booking-service unreachable")})

	categories, err := svc.GetSeatMap(store.events[0].ID.String())
	if err != nil {
		t.Fatalf("GetSeatMap = %v, want degradation instead of an error", err)
	}
	if len(categories) != 1 || len(categories[0].Rows) != 1 {
		t.Fatalf("categories/rows = %d/%d, want 1/1", len(categories), len(categories[0].Rows))
	}

	// Every seat falls back to "available" while booking-service is down.
	for _, seat := range append(categories[0].Rows[0].LeftSeats, categories[0].Rows[0].RightSeats...) {
		if seat.Status != "available" {
			t.Errorf("seat %s status = %q, want available", seat.SeatID, seat.Status)
		}
	}
}

func TestGetSeatMapAppliesLiveBookingStatuses(t *testing.T) {
	store := seedStore(uuid.New())
	seats := store.seats[store.events[0].ID]
	booking := &fakeBookingClient{statuses: []client.SeatStatus{
		{SeatID: seats[0].ID.String(), Status: "CONFIRMED"},
		{SeatID: seats[1].ID.String(), Status: "LOCKED"},
	}}
	svc := newTestService(store, booking)

	categories, err := svc.GetSeatMap(store.events[0].ID.String())
	if err != nil {
		t.Fatalf("GetSeatMap: %v", err)
	}

	row := categories[0].Rows[0]
	if got := row.LeftSeats[0].Status; got != "sold" {
		t.Errorf("CONFIRMED status = %q, want sold", got)
	}
	if got := row.RightSeats[0].Status; got != "locked" {
		t.Errorf("LOCKED status = %q, want locked", got)
	}
}

// validCreateRequest is the happy-path payload used by the tests below.
func validCreateRequest() *dto.CreateEventRequest {
	return &dto.CreateEventRequest{
		Name:      "Konser Nusantara 2026",
		Venue:     "GBK Jakarta",
		EventDate: "2026-06-15T19:30:00+07:00",
		Categories: []dto.CreateEventCategory{
			{Name: "VIP", Price: 1500000},
			{Name: "Festival", Price: 900000},
		},
	}
}

func TestCreateEventGeneratesSeatmap(t *testing.T) {
	store := &fakeCatalogStore{}
	svc := newTestService(store, nil)
	caller := authz.Caller{UserID: uuid.New().String()}

	event, err := svc.CreateEvent(validCreateRequest(), caller)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if event.Code != "konser-nusantara-2026" {
		t.Errorf("code = %q, want slug of the name", event.Code)
	}
	if event.CreatedBy == nil || event.CreatedBy.String() != caller.UserID {
		t.Errorf("createdBy = %v, want the caller %s", event.CreatedBy, caller.UserID)
	}
	wantDate := time.Date(2026, 6, 15, 19, 30, 0, 0, time.FixedZone("WIB", 7*3600))
	if !event.EventDate.Equal(wantDate) {
		t.Errorf("eventDate = %v, want %v", event.EventDate, wantDate)
	}

	if store.created == nil {
		t.Fatal("store received no event")
	}
	if len(store.created.categories) != 2 {
		t.Fatalf("categories = %d, want 2", len(store.created.categories))
	}
	// 2 categories x 4 rows (A-D) x 16 seats.
	if len(store.created.seats) != 128 {
		t.Fatalf("seats = %d, want 128", len(store.created.seats))
	}

	first := store.created.seats[0]
	if first.Label != "C1-A01" || first.RowName != "A" || first.Number != 1 || first.Side != "left" {
		t.Errorf("first seat = %+v, want C1-A01 on the left", first)
	}
	if first.SortOrder != 1 {
		t.Errorf("sortOrder = %d, want 1", first.SortOrder)
	}
	if first.EventID != event.ID || first.CategoryID != store.created.categories[0].ID {
		t.Errorf("seat %s: event/category = %s/%s, want %s/%s",
			first.Label, first.EventID, first.CategoryID, event.ID, store.created.categories[0].ID)
	}

	// Seat 9 switches to the right side, mirroring the seed layout.
	right := store.created.seats[8]
	if right.Number != 9 || right.Side != "right" {
		t.Errorf("seat 9 = %+v, want side right", right)
	}
	// Last seat of the last row: D16 on the right.
	last := store.created.seats[len(store.created.seats)-1]
	if last.Label != "C2-D16" || last.Side != "right" || last.SortOrder != 316 {
		t.Errorf("last seat = %+v, want C2-D16 sortOrder 316", last)
	}

	if store.created.categories[0].Code != "konser-nusantara-2026-1" {
		t.Errorf("category code = %q, want %q", store.created.categories[0].Code, "konser-nusantara-2026-1")
	}
	if store.created.categories[1].Price != 900000 || store.created.categories[1].SortOrder != 2 {
		t.Errorf("second category = %+v, want Festival at 900000 order 2", store.created.categories[1])
	}
}

func TestCreateEventSupportsCustomSeatLayoutAndPayout(t *testing.T) {
	store := &fakeCatalogStore{}
	svc := newTestService(store, nil)

	req := validCreateRequest()
	req.SeatRows = 3
	req.SeatsPerRow = 12
	req.LeftBlockSize = 5
	req.PayoutBankName = "BCA"
	req.PayoutAccountNumber = "1234567890"
	req.PayoutAccountName = "PT Nusantara Konser"

	event, err := svc.CreateEvent(req, authz.Caller{UserID: uuid.New().String()})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	if event.SeatRows != 3 || event.SeatsPerRow != 12 || event.LeftBlockSize != 5 {
		t.Errorf("seat layout = %d/%d/%d, want 3/12/5", event.SeatRows, event.SeatsPerRow, event.LeftBlockSize)
	}
	if event.PayoutBankName != "BCA" || event.PayoutAccountNumber != "1234567890" || event.PayoutAccountName != "PT Nusantara Konser" {
		t.Errorf("payout fields = %q/%q/%q, want BCA/1234567890/PT Nusantara Konser", event.PayoutBankName, event.PayoutAccountNumber, event.PayoutAccountName)
	}

	if store.created == nil {
		t.Fatal("store received no event")
	}
	if len(store.created.seats) != 72 {
		t.Fatalf("seats = %d, want 72", len(store.created.seats))
	}
	if got := store.created.seats[5].Side; got != "right" {
		t.Errorf("seat 6 side = %q, want right (left block only 1-5)", got)
	}
}

func TestCreateEventDerivesUniqueCode(t *testing.T) {
	taken := uuid.New()
	store := &fakeCatalogStore{events: []model.Event{{ID: taken, Code: "konser-nusantara-2026"}}}
	svc := newTestService(store, nil)

	event, err := svc.CreateEvent(validCreateRequest(), authz.Caller{UserID: uuid.New().String()})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if event.Code != "konser-nusantara-2026-2" {
		t.Errorf("code = %q, want the -2 suffix", event.Code)
	}
}

func TestCreateEventRejectsDuplicateRequestedCode(t *testing.T) {
	store := seedStore(uuid.New()) // seeded event uses code COLDPLAY
	svc := newTestService(store, nil)

	req := validCreateRequest()
	req.Code = "COLDPLAY"

	_, err := svc.CreateEvent(req, authz.Caller{UserID: uuid.New().String()})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateEvent = %v, want ErrInvalidInput", err)
	}
}

func TestCreateEventValidatesInput(t *testing.T) {
	caller := authz.Caller{UserID: uuid.New().String()}

	tests := []struct {
		name   string
		mutate func(*dto.CreateEventRequest)
	}{
		{"bad date", func(r *dto.CreateEventRequest) { r.EventDate = "kemarin" }},
		{"empty name", func(r *dto.CreateEventRequest) { r.Name = "   " }},
		{"empty venue", func(r *dto.CreateEventRequest) { r.Venue = "" }},
		{"no categories", func(r *dto.CreateEventRequest) { r.Categories = nil }},
		{"empty category name", func(r *dto.CreateEventRequest) { r.Categories[0].Name = " " }},
		{"negative price", func(r *dto.CreateEventRequest) { r.Categories[0].Price = -1 }},
		{"left block too large", func(r *dto.CreateEventRequest) { r.LeftBlockSize = 16; r.SeatsPerRow = 16 }},
		{"incomplete payout fields", func(r *dto.CreateEventRequest) { r.PayoutBankName = "BCA" }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeCatalogStore{}
			svc := newTestService(store, nil)
			req := validCreateRequest()
			tc.mutate(req)

			event, err := svc.CreateEvent(req, caller)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("CreateEvent = %v, want ErrInvalidInput", err)
			}
			if event != nil {
				t.Errorf("event = %+v, want nil", event)
			}
			if store.created != nil {
				t.Error("store persisted a rejected event")
			}
		})
	}
}

func TestCreateEventAcceptsPlainDate(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{}, nil)

	req := validCreateRequest()
	req.EventDate = "2026-06-15"

	event, err := svc.CreateEvent(req, authz.Caller{UserID: uuid.New().String()})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if event.EventDate.Day() != 15 || event.EventDate.Month() != time.June {
		t.Errorf("eventDate = %v, want 15 June 2026", event.EventDate)
	}
}

func TestCreateEventRequiresSubject(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{}, nil)

	_, err := svc.CreateEvent(validCreateRequest(), authz.Caller{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateEvent = %v, want ErrInvalidInput for an empty subject", err)
	}
}

func TestCreateEventPropagatesStoreError(t *testing.T) {
	boom := errors.New("db down")
	store := &fakeCatalogStore{createErr: boom}
	svc := newTestService(store, nil)

	if _, err := svc.CreateEvent(validCreateRequest(), authz.Caller{UserID: uuid.New().String()}); !errors.Is(err, boom) {
		t.Fatalf("CreateEvent = %v, want the store error", err)
	}
}

// --- discovery & tier sale mode -------------------------------------------

func tierCreateRequest() *dto.CreateEventRequest {
	return &dto.CreateEventRequest{
		Name:      "Tech Summit 2027",
		Venue:     "ICE BSD",
		City:      "Tangerang",
		Category:  "seminar",
		EventDate: "2027-03-05T09:00:00+07:00",
		SaleMode:  "TIER",
		Categories: []dto.CreateEventCategory{
			{Name: "VVIP", Price: 1500000, Quota: 50, Description: "Front row"},
			{Name: "VIP", Price: 750000, Quota: 150},
			{Name: "Regular", Price: 350000, Quota: 500},
		},
	}
}

func TestCreateEventTierModeGeneratesNoSeats(t *testing.T) {
	store := &fakeCatalogStore{}
	svc := newTestService(store, nil)

	event, err := svc.CreateEvent(tierCreateRequest(), authz.Caller{UserID: uuid.New().String()})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	if event.SaleMode != model.SaleModeTier {
		t.Errorf("saleMode = %q, want TIER", event.SaleMode)
	}
	if event.City != "Tangerang" || event.Category != "seminar" {
		t.Errorf("city/category = %q/%q, want Tangerang/seminar", event.City, event.Category)
	}
	if store.created == nil {
		t.Fatal("store received no event")
	}
	if len(store.created.seats) != 0 {
		t.Errorf("seats = %d, want 0 for a TIER event", len(store.created.seats))
	}
	if len(store.created.categories) != 3 {
		t.Fatalf("categories = %d, want 3", len(store.created.categories))
	}
	for _, category := range store.created.categories {
		if category.Quota < 1 {
			t.Errorf("category %q quota = %d, want >= 1", category.Name, category.Quota)
		}
	}
}

func TestCreateEventTierRequiresQuota(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{}, nil)

	req := tierCreateRequest()
	req.Categories[1].Quota = 0 // VIP tanpa kuota

	_, err := svc.CreateEvent(req, authz.Caller{UserID: uuid.New().String()})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateEvent = %v, want ErrInvalidInput", err)
	}
}

func TestCreateEventRejectsSeatedWithQuota(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{}, nil)

	req := validCreateRequest()
	req.SaleMode = "SEATED"
	req.Categories[0].Quota = 10

	_, err := svc.CreateEvent(req, authz.Caller{UserID: uuid.New().String()})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateEvent = %v, want ErrInvalidInput", err)
	}
}

func TestCreateEventInfersTierFromQuota(t *testing.T) {
	store := &fakeCatalogStore{}
	svc := newTestService(store, nil)

	req := validCreateRequest()
	for i := range req.Categories {
		req.Categories[i].Quota = 100
	}

	event, err := svc.CreateEvent(req, authz.Caller{UserID: uuid.New().String()})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if event.SaleMode != model.SaleModeTier {
		t.Errorf("saleMode = %q, want TIER inferred from the quota", event.SaleMode)
	}
}

func TestCreateEventValidatesDiscoveryMetadata(t *testing.T) {
	caller := authz.Caller{UserID: uuid.New().String()}

	tests := []struct {
		name   string
		mutate func(*dto.CreateEventRequest)
	}{
		{"unknown category", func(r *dto.CreateEventRequest) { r.Category = "kopi" }},
		{"unknown sale mode", func(r *dto.CreateEventRequest) { r.SaleMode = "RAFFLE" }},
		{"poster not http", func(r *dto.CreateEventRequest) { r.PosterURL = "ftp://x" }},
		{"oversized description", func(r *dto.CreateEventRequest) { r.Description = strings.Repeat("a", 2001) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeCatalogStore{}
			svc := newTestService(store, nil)
			req := tierCreateRequest()
			tc.mutate(req)

			if _, err := svc.CreateEvent(req, caller); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("CreateEvent = %v, want ErrInvalidInput", err)
			}
			if store.created != nil {
				t.Error("store persisted a rejected event")
			}
		})
	}
}

func TestGetTicketTypesComputesRemaining(t *testing.T) {
	store := seedStore(uuid.New())
	eventID := store.events[0].ID
	store.events[0].SaleMode = model.SaleModeTier
	store.known[eventID].SaleMode = model.SaleModeTier

	// Booking reports 5 of the 10 VIP tickets reserved.
	categoryID := store.categories[eventID][0].ID
	store.categories[eventID][0].Quota = 10

	booking := &fakeBookingClient{tickets: []client.TicketStatus{
		{CategoryID: categoryID.String(), Reserved: 5},
	}}
	svc := newTestService(store, booking)

	response, err := svc.GetTicketTypes(eventID.String())
	if err != nil {
		t.Fatalf("GetTicketTypes: %v", err)
	}
	if response.SaleMode != string(model.SaleModeTier) {
		t.Errorf("saleMode = %q, want TIER", response.SaleMode)
	}
	if len(response.TicketTypes) != 1 {
		t.Fatalf("ticketTypes = %d, want 1", len(response.TicketTypes))
	}
	if got := response.TicketTypes[0].Remaining; got != 5 {
		t.Errorf("remaining = %d, want 5", got)
	}
	if got := response.TicketTypes[0].Quota; got != 10 {
		t.Errorf("quota = %d, want 10", got)
	}
}

func TestGetTicketTypesDegradesWhenBookingIsDown(t *testing.T) {
	store := seedStore(uuid.New())
	store.events[0].SaleMode = model.SaleModeTier
	store.known[store.events[0].ID].SaleMode = model.SaleModeTier
	store.categories[store.events[0].ID][0].Quota = 20
	svc := newTestService(store, &fakeBookingClient{err: errors.New("booking-service unreachable")})

	response, err := svc.GetTicketTypes(store.events[0].ID.String())
	if err != nil {
		t.Fatalf("GetTicketTypes: %v, want graceful degradation", err)
	}
	// No availability feed: the tier still lists (optimistic, same as seats
	// degrading to "available") and checkout itself still requires booking.
	if got := response.TicketTypes[0].Remaining; got != 20 {
		t.Errorf("remaining = %d, want the full quota while booking-service is down", got)
	}
}

func TestGetTicketTypesUnknownEvent(t *testing.T) {
	svc := newTestService(&fakeCatalogStore{known: map[uuid.UUID]*model.Event{}}, nil)

	if _, err := svc.GetTicketTypes(uuid.NewString()); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("err = %v, want ErrEventNotFound", err)
	}
	if _, err := svc.GetTicketTypes("nope"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestListCities(t *testing.T) {
	store := &fakeCatalogStore{events: []model.Event{
		{ID: uuid.New(), City: "Jakarta"},
		{ID: uuid.New(), City: "Jakarta"},
		{ID: uuid.New(), City: "Bandung"},
	}}
	svc := newTestService(store, nil)

	cities, err := svc.ListCities()
	if err != nil {
		t.Fatalf("ListCities: %v", err)
	}
	if len(cities) != 2 {
		t.Fatalf("cities = %d, want 2", len(cities))
	}
	counts := map[string]int{}
	for _, entry := range cities {
		counts[entry.City] = entry.Count
	}
	if counts["Jakarta"] != 2 || counts["Bandung"] != 1 {
		t.Errorf("counts = %v, want Jakarta=2 Bandung=1", counts)
	}
}
