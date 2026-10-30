package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeEventStore satisfies service's consumer-side eventStore interface.
// categoryID is shared so seats group under the category they belong to.
type fakeEventStore struct {
	categoryID uuid.UUID
	categories map[uuid.UUID][]model.SeatCategory
	events     []model.Event
	eventsErr  error
	known      map[uuid.UUID]*model.Event
	lookupErr  error
	createErr  error
	created    *model.Event
}

func (f *fakeEventStore) FindEvents(filter dto.EventFilter) ([]model.Event, error) {
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
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

func (f *fakeEventStore) FindEventCount(filter dto.EventFilter) (int64, error) {
	if f.eventsErr != nil {
		return 0, f.eventsErr
	}
	events, err := f.FindEvents(filter)
	if err != nil {
		return 0, err
	}
	return int64(len(events)), nil
}

func (f *fakeEventStore) FindDistinctCities() ([]dto.CityEntry, error) {
	return []dto.CityEntry{{City: "Jakarta", Count: len(f.events)}}, nil
}

func (f *fakeEventStore) FindCategoriesByEvents([]uuid.UUID) ([]model.SeatCategory, error) {
	return f.FindCategoriesByEvent(uuid.Nil)
}

func (f *fakeEventStore) FindByID(id uuid.UUID) (*model.Event, error) {
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

func (f *fakeEventStore) FindCategoriesByEvent(eventID uuid.UUID) ([]model.SeatCategory, error) {
	if f.categories != nil {
		return f.categories[eventID], nil
	}
	return []model.SeatCategory{{ID: f.categoryID, Code: "CAT-A", Name: "VIP", Price: 100}}, nil
}

func (f *fakeEventStore) FindSeatsByEvent(uuid.UUID) ([]model.Seat, error) {
	return []model.Seat{{
		ID: uuid.New(), CategoryID: f.categoryID, Label: "A1", RowName: "A", Number: 1, Side: "left",
	}}, nil
}

func (f *fakeEventStore) CodeExists(code string) (bool, error) {
	for _, event := range f.events {
		if strings.EqualFold(event.Code, code) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeEventStore) CreateEventWithSeatmap(event *model.Event, categories []model.SeatCategory, seats []model.Seat) error {
	if f.createErr != nil {
		return f.createErr
	}
	stored := *event
	f.created = &stored
	f.events = append(f.events, stored)
	if f.known == nil {
		f.known = map[uuid.UUID]*model.Event{}
	}
	f.known[event.ID] = &stored
	if f.categories == nil {
		f.categories = map[uuid.UUID][]model.SeatCategory{}
	}
	f.categories[event.ID] = categories
	return nil
}

type fakeBookingClient struct {
	statuses []client.SeatStatus
	tickets  []client.TicketStatus
	err      error
}

func (f *fakeBookingClient) GetSeatStatus(string) ([]client.SeatStatus, error) {
	return f.statuses, f.err
}

func (f *fakeBookingClient) GetTicketStatus(string) ([]client.TicketStatus, error) {
	return f.tickets, f.err
}

// testJWTSecret signs the access tokens the handler tests authenticate with.
const testJWTSecret = "test-only-catalog-secret"

// testServiceToken is the shared X-Service-Token the internal tier-quotas
// feed demands.
const testServiceToken = "test-service-token"

// newRouter mirrors main.go's route table, including the RequireUser guard on
// POST /events and the RequireService guard on the internal stock feed.
func newRouter(store *fakeEventStore, booking client.BookingClient) *gin.Engine {
	svc := service.NewCatalogService(store, booking)
	h := NewCatalogHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	api.GET("/events", h.ListEvents)
	api.POST("/events", authz.RequireUser(testJWTSecret), h.CreateEvent)
	api.GET("/events/cities", h.ListCities)
	api.GET("/events/:id", h.GetEvent)
	api.GET("/events/:id/seatmap", h.GetSeatMap)
	api.GET("/events/:id/ticket-types", h.GetTicketTypes)
	api.GET("/events/:id/tier-quotas", authz.RequireService(testServiceToken), h.GetTierQuotas)
	return r
}

// accessTokenFor mints a valid HS256 access JWT for the given subject, the
// way auth-service signs them.
func accessTokenFor(t *testing.T, subject string) string {
	t.Helper()
	claims := jwtlib.MapClaims{
		"sub":   subject,
		"email": "poster@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func post(t *testing.T, r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, uuid.New().String()))
	r.ServeHTTP(w, req)
	return w
}

func get(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return body
}

func wantMessage(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	body := decode(t, w)
	got, _ := body["message"].(string)
	if got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
	if len(body) != 1 {
		t.Errorf("error envelope = %v, want only the message key", body)
	}
}

func TestListEventsHandler(t *testing.T) {
	eventID := uuid.New()
	store := &fakeEventStore{events: []model.Event{{ID: eventID, Code: "COLDPLAY", Name: "Coldplay", Venue: "GBK"}}}

	w := get(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	body := decode(t, w)
	events, ok := body["events"].([]any)
	if !ok {
		t.Fatalf("body = %v, want an events array", body)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	first, _ := events[0].(map[string]any)
	if first["id"] != eventID.String() || first["code"] != "COLDPLAY" {
		t.Errorf("event = %v, want id %s and code COLDPLAY", first, eventID)
	}
}

func TestListEventsHandlerEmpty(t *testing.T) {
	w := get(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if events := decode(t, w)["events"].([]any); len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
}

func TestListEventsHandlerStoreError(t *testing.T) {
	store := &fakeEventStore{eventsErr: errors.New("db down")}

	w := get(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	wantMessage(t, w, "db down")
}

func TestGetSeatMapHandler(t *testing.T) {
	eventID := uuid.New()
	store := &fakeEventStore{categoryID: uuid.New(), known: map[uuid.UUID]*model.Event{eventID: {ID: eventID, Name: "Coldplay"}}}

	w := get(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events/"+eventID.String()+"/seatmap")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	body := decode(t, w)
	categories, ok := body["categories"].([]any)
	if !ok {
		t.Fatalf("body = %v, want a categories array", body)
	}
	if len(categories) != 1 {
		t.Fatalf("categories = %d, want 1", len(categories))
	}
	category, _ := categories[0].(map[string]any)
	rows, _ := category["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	row, _ := rows[0].(map[string]any)
	left, _ := row["leftSeats"].([]any)
	if len(left) != 1 {
		t.Fatalf("leftSeats = %d, want 1", len(left))
	}
	seat, _ := left[0].(map[string]any)
	if seat["status"] != "available" {
		t.Errorf("status = %v, want available", seat["status"])
	}
}

func TestGetSeatMapHandlerErrors(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		w := get(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events/not-a-uuid/seatmap")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantMessage(t, w, "invalid input: eventId")
	})

	t.Run("unknown event", func(t *testing.T) {
		w := get(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events/"+uuid.New().String()+"/seatmap")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		wantMessage(t, w, "event not found")
	})

	t.Run("store failure", func(t *testing.T) {
		store := &fakeEventStore{lookupErr: errors.New("db down")}
		w := get(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events/"+uuid.New().String()+"/seatmap")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
		wantMessage(t, w, "db down")
	})
}

// TestGetSeatMapHandlerDegradesWithoutBookingClient proves the seatmap still
// renders (200 + available seats) when booking-service is unreachable.
func TestGetSeatMapHandlerDegradesWithoutBookingClient(t *testing.T) {
	eventID := uuid.New()
	store := &fakeEventStore{known: map[uuid.UUID]*model.Event{eventID: {ID: eventID, Name: "Coldplay"}}}
	booking := &fakeBookingClient{err: errors.New("booking-service unreachable")}

	w := get(t, newRouter(store, booking), "/api/v1/events/"+eventID.String()+"/seatmap")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if categories := decode(t, w)["categories"].([]any); len(categories) != 1 {
		t.Errorf("categories = %d, want 1", len(categories))
	}
}

const createEventBody = `{
	"name": "Konser Nusantara 2026",
	"venue": "GBK Jakarta",
	"eventDate": "2026-06-15T19:30:00+07:00",
	"categories": [
		{"name": "VIP", "price": 1500000},
		{"name": "Festival", "price": 900000}
	]
}`

func TestCreateEventHandler(t *testing.T) {
	store := &fakeEventStore{}

	w := post(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events", createEventBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	body := decode(t, w)
	if body["code"] != "konser-nusantara-2026" || body["name"] != "Konser Nusantara 2026" {
		t.Errorf("event = %v, want the posted name and its slug", body)
	}
	if body["venue"] != "GBK Jakarta" {
		t.Errorf("venue = %v, want GBK Jakarta", body["venue"])
	}
	if body["createdBy"] == nil || body["createdBy"] == "" {
		t.Error("createdBy missing: the poster must be recorded")
	}
	if store.created == nil {
		t.Fatal("store received no event")
	}
	if store.created.ID.String() != body["id"] {
		t.Errorf("id = %v, want %s", body["id"], store.created.ID)
	}
}

func TestCreateEventHandlerErrors(t *testing.T) {
	t.Run("malformed body", func(t *testing.T) {
		w := post(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events", `{"name":`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		w := post(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events", `{"name": "Konser"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
		}
	})

	t.Run("duplicate code", func(t *testing.T) {
		store := &fakeEventStore{events: []model.Event{{ID: uuid.New(), Code: "konser-nusantara-2026"}}}
		w := post(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events",
			`{"code":"konser-nusantara-2026","name":"Lain","venue":"GBK","eventDate":"2026-06-15","categories":[{"name":"VIP","price":1}]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
		}
		wantMessage(t, w, `invalid input: kode event "konser-nusantara-2026" sudah dipakai`)
	})

	t.Run("bad date", func(t *testing.T) {
		store := &fakeEventStore{}
		w := post(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events",
			`{"name":"Lain","venue":"GBK","eventDate":"kemarin","categories":[{"name":"VIP","price":1}]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("store failure", func(t *testing.T) {
		store := &fakeEventStore{createErr: errors.New("db down")}
		w := post(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events", createEventBody)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
		wantMessage(t, w, "db down")
	})

	t.Run("without access token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(createEventBody))
		req.Header.Set("Content-Type", "application/json")
		newRouter(&fakeEventStore{}, &fakeBookingClient{}).ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body = %s", w.Code, w.Body.String())
		}
	})
}

// --- discovery, detail and tier endpoints ----------------------------------

func TestListEventsHandlerFilters(t *testing.T) {
	jakarta := uuid.New()
	bandung := uuid.New()
	store := &fakeEventStore{events: []model.Event{
		{ID: jakarta, Code: "jazz", Name: "Java Jazz", Venue: "JIExpo", City: "Jakarta", Category: "festival"},
		{ID: bandung, Code: "rock", Name: "Bandung Rock", Venue: "Sabilulungan", City: "Bandung", Category: "concert"},
	}}
	router := newRouter(store, &fakeBookingClient{})

	t.Run("by city", func(t *testing.T) {
		w := get(t, router, "/api/v1/events?city=Bandung")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := decode(t, w)
		events := body["events"].([]any)
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1", len(events))
		}
		if got := events[0].(map[string]any)["id"]; got != bandung.String() {
			t.Errorf("id = %v, want %s", got, bandung)
		}
		if body["total"].(float64) != 1 {
			t.Errorf("total = %v, want 1", body["total"])
		}
	})

	t.Run("by search text", func(t *testing.T) {
		w := get(t, router, "/api/v1/events?q=jazz")
		events := decode(t, w)["events"].([]any)
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1", len(events))
		}
		if got := events[0].(map[string]any)["name"]; got != "Java Jazz" {
			t.Errorf("name = %v, want Java Jazz", got)
		}
	})

	t.Run("by category", func(t *testing.T) {
		w := get(t, router, "/api/v1/events?category=concert")
		events := decode(t, w)["events"].([]any)
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1", len(events))
		}
		if got := events[0].(map[string]any)["city"]; got != "Bandung" {
			t.Errorf("city = %v, want Bandung", got)
		}
	})
}

func TestListCitiesHandler(t *testing.T) {
	store := &fakeEventStore{events: []model.Event{{ID: uuid.New(), City: "Jakarta"}}}

	w := get(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events/cities")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	cities := decode(t, w)["cities"].([]any)
	if len(cities) != 1 {
		t.Fatalf("cities = %d, want 1", len(cities))
	}
	if got := cities[0].(map[string]any)["city"]; got != "Jakarta" {
		t.Errorf("city = %v, want Jakarta", got)
	}
}

func TestGetEventHandler(t *testing.T) {
	eventID := uuid.New()
	store := &fakeEventStore{known: map[uuid.UUID]*model.Event{
		eventID: {ID: eventID, Code: "tech-summit", Name: "Tech Summit", Venue: "ICE BSD", City: "Tangerang", Category: "seminar", SaleMode: model.SaleModeTier},
	}}
	store.categoryID = uuid.New()

	w := get(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events/"+eventID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["name"] != "Tech Summit" || body["city"] != "Tangerang" || body["saleMode"] != "TIER" {
		t.Errorf("event = %v, want the stored discovery fields", body)
	}
	if body["minPrice"] != float64(100) {
		t.Errorf("minPrice = %v, want 100", body["minPrice"])
	}

	t.Run("unknown event", func(t *testing.T) {
		w := get(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events/"+uuid.NewString())
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		w := get(t, newRouter(&fakeEventStore{}, &fakeBookingClient{}), "/api/v1/events/nope")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestGetTicketTypesHandler(t *testing.T) {
	eventID := uuid.New()
	categoryID := uuid.New()
	store := &fakeEventStore{
		categoryID: categoryID,
		known:      map[uuid.UUID]*model.Event{eventID: {ID: eventID, Name: "Tech Summit", SaleMode: model.SaleModeTier}},
	}
	store.categories = map[uuid.UUID][]model.SeatCategory{
		eventID: {{ID: categoryID, EventID: eventID, Code: "T1", Name: "VVIP", Price: 1500000, Quota: 50}},
	}
	booking := &fakeBookingClient{tickets: []client.TicketStatus{{CategoryID: categoryID.String(), Reserved: 8}}}

	w := get(t, newRouter(store, booking), "/api/v1/events/"+eventID.String()+"/ticket-types")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	types := body["ticketTypes"].([]any)
	if len(types) != 1 {
		t.Fatalf("ticketTypes = %d, want 1", len(types))
	}
	first := types[0].(map[string]any)
	if first["quota"] != float64(50) || first["remaining"] != float64(42) {
		t.Errorf("tier = %v, want quota 50 remaining 42", first)
	}
}

func TestGetTierQuotasHandlerRequiresServiceToken(t *testing.T) {
	eventID := uuid.New()
	store := &fakeEventStore{known: map[uuid.UUID]*model.Event{eventID: {ID: eventID, Name: "Tech Summit"}}}
	store.categories = map[uuid.UUID][]model.SeatCategory{
		eventID: {{ID: uuid.New(), EventID: eventID, Code: "T1", Name: "VIP", Price: 750000, Quota: 150}},
	}
	router := newRouter(store, &fakeBookingClient{})

	t.Run("without token", func(t *testing.T) {
		w := get(t, router, "/api/v1/events/"+eventID.String()+"/tier-quotas")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body = %s", w.Code, w.Body.String())
		}
	})

	t.Run("with token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+eventID.String()+"/tier-quotas", nil)
		req.Header.Set("X-Service-Token", testServiceToken)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
		}
		tiers := decode(t, w)["tiers"].([]any)
		if len(tiers) != 1 {
			t.Fatalf("tiers = %d, want 1", len(tiers))
		}
		first := tiers[0].(map[string]any)
		if first["quota"] != float64(150) || first["price"] != float64(750000) {
			t.Errorf("tier = %v, want quota 150 price 750000", first)
		}
	})
}

const createTierEventBody = `{
	"name": "Tech Summit 2027",
	"venue": "ICE BSD",
	"city": "Tangerang",
	"category": "seminar",
	"saleMode": "TIER",
	"eventDate": "2027-03-05T09:00:00+07:00",
	"categories": [
		{"name": "VVIP", "price": 1500000, "quota": 50},
		{"name": "VIP", "price": 750000, "quota": 150},
		{"name": "Regular", "price": 350000, "quota": 500}
	]
}`

func TestCreateTierEventHandler(t *testing.T) {
	store := &fakeEventStore{}

	w := post(t, newRouter(store, &fakeBookingClient{}), "/api/v1/events", createTierEventBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	body := decode(t, w)
	if body["saleMode"] != "TIER" || body["city"] != "Tangerang" || body["category"] != "seminar" {
		t.Errorf("event = %v, want TIER/Tangerang/seminar", body)
	}
	if body["minPrice"] != float64(350000) {
		t.Errorf("minPrice = %v, want 350000", body["minPrice"])
	}
	if body["ticketTypeCount"] != float64(3) {
		t.Errorf("ticketTypeCount = %v, want 3", body["ticketTypeCount"])
	}
}
