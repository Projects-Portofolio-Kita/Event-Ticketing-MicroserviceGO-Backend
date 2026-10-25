package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeEventStore satisfies service's consumer-side eventStore interface.
// categoryID is shared so seats group under the category they belong to.
type fakeEventStore struct {
	categoryID uuid.UUID
	events     []model.Event
	eventsErr  error
	known      map[uuid.UUID]*model.Event
	lookupErr  error
}

func (f *fakeEventStore) FindEvents() ([]model.Event, error) {
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
	return f.events, nil
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

func (f *fakeEventStore) FindCategoriesByEvent(uuid.UUID) ([]model.SeatCategory, error) {
	return []model.SeatCategory{{ID: f.categoryID, Code: "CAT-A", Name: "VIP", Price: 100}}, nil
}

func (f *fakeEventStore) FindSeatsByEvent(uuid.UUID) ([]model.Seat, error) {
	return []model.Seat{{
		ID: uuid.New(), CategoryID: f.categoryID, Label: "A1", RowName: "A", Number: 1, Side: "left",
	}}, nil
}

type fakeBookingClient struct {
	statuses []client.SeatStatus
	err      error
}

func (f *fakeBookingClient) GetSeatStatus(string) ([]client.SeatStatus, error) {
	return f.statuses, f.err
}

// newRouter mirrors main.go's route table.
func newRouter(store *fakeEventStore, booking client.BookingClient) *gin.Engine {
	svc := service.NewCatalogService(store, booking)
	h := NewCatalogHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	api.GET("/events", h.ListEvents)
	api.GET("/events/:id/seatmap", h.GetSeatMap)
	return r
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
