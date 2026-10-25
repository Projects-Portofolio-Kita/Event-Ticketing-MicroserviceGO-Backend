package service

import (
	"errors"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
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
}

func (f *fakeCatalogStore) FindEvents() ([]model.Event, error) {
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
	return f.events, nil
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

// fakeBookingClient stands in for the booking-service availability feed.
type fakeBookingClient struct {
	statuses []client.SeatStatus
	err      error
}

func (f *fakeBookingClient) GetSeatStatus(eventID string) ([]client.SeatStatus, error) {
	return f.statuses, f.err
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

	events, err := svc.ListEvents()
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
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

	events, err := svc.ListEvents()
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if events == nil {
		t.Fatal("events = nil, want an empty non-nil slice")
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
}

func TestListEventsPropagatesStoreError(t *testing.T) {
	boom := errors.New("db down")
	svc := newTestService(&fakeCatalogStore{eventsErr: boom}, nil)

	if _, err := svc.ListEvents(); !errors.Is(err, boom) {
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
