package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// recorded is one request captured by the fake notification-service.
type recorded struct {
	method string
	path   string
	header http.Header
	body   []byte
}

// recorder collects requests without racing the handler goroutine.
type recorder struct {
	mu   sync.Mutex
	reqs []recorded
}

func (r *recorder) add(req recorded) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
}

func (r *recorder) all() []recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recorded(nil), r.reqs...)
}

// newNotificationServer answers with the given status and records every
// request it receives.
func newNotificationServer(t *testing.T, status int) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.add(recorded{method: r.Method, path: r.URL.Path, header: r.Header.Clone(), body: body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// mustNotPanic fails the test when fn panics: a notification must never break
// the calling business flow.
func mustNotPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", what, r)
		}
	}()
	fn()
}

func sampleMessage() Message {
	return Message{
		UserID:    "11111111-1111-1111-1111-111111111111",
		Kind:      KindBookingConfirmed,
		Title:     "Booking confirmed",
		Body:      "Your booking is confirmed",
		Reference: "BKG-42",
	}
}

func singleRequest(t *testing.T, rec *recorder) recorded {
	t.Helper()
	reqs := rec.all()
	if len(reqs) != 1 {
		t.Fatalf("captured %d requests, want 1", len(reqs))
	}
	return reqs[0]
}

func TestNewReadsEnvAndPostsMessage(t *testing.T) {
	srv, rec := newNotificationServer(t, http.StatusCreated)
	t.Setenv("NOTIFICATION_SERVICE_URL", srv.URL)
	t.Setenv("SERVICE_TOKEN", "internal-secret")

	mustNotPanic(t, "Send with configured env", func() { New().Send(sampleMessage()) })

	got := singleRequest(t, rec)
	if got.method != http.MethodPost {
		t.Errorf("method = %s, want %s", got.method, http.MethodPost)
	}
	if got.path != "/api/v1/notifications" {
		t.Errorf("path = %q, want %q", got.path, "/api/v1/notifications")
	}
	if ct := got.header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if token := got.header.Get("X-Service-Token"); token != "internal-secret" {
		t.Errorf("X-Service-Token = %q, want %q", token, "internal-secret")
	}

	var payload Message
	if err := json.Unmarshal(got.body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload != sampleMessage() {
		t.Errorf("payload = %+v, want %+v", payload, sampleMessage())
	}
}

func TestNewWithBaseURLTrimsTrailingSlash(t *testing.T) {
	srv, rec := newNotificationServer(t, http.StatusCreated)

	NewWithBaseURL(srv.URL+"/", "").Send(sampleMessage())

	got := singleRequest(t, rec)
	if got.path != "/api/v1/notifications" {
		t.Errorf("path = %q, want %q", got.path, "/api/v1/notifications")
	}
}

func TestServiceTokenHeaderDependsOnConfig(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantHeader bool
	}{
		{"configured token is sent", "internal-secret", true},
		{"empty token omits the header", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rec := newNotificationServer(t, http.StatusOK)

			NewWithBaseURL(srv.URL, tt.token).Send(sampleMessage())

			got := singleRequest(t, rec)
			_, present := got.header["X-Service-Token"]
			if !tt.wantHeader {
				if present {
					t.Error("X-Service-Token must not be sent when the token is empty")
				}
				return
			}
			if !present {
				t.Fatal("X-Service-Token header is missing")
			}
			if token := got.header.Get("X-Service-Token"); token != tt.token {
				t.Errorf("X-Service-Token = %q, want %q", token, tt.token)
			}
		})
	}
}

func TestNewWithoutEnvURLIsNoOp(t *testing.T) {
	_, rec := newNotificationServer(t, http.StatusCreated)
	t.Setenv("NOTIFICATION_SERVICE_URL", "")
	t.Setenv("SERVICE_TOKEN", "internal-secret")

	mustNotPanic(t, "Send without a base URL", func() { New().Send(sampleMessage()) })

	if n := len(rec.all()); n != 0 {
		t.Errorf("sent %d requests with an empty URL, want 0", n)
	}
}

func TestNoopDiscardsMessages(t *testing.T) {
	n := Noop()
	if _, ok := n.(noop); !ok {
		t.Errorf("Noop() returned %T, want the noop notifier", n)
	}
	mustNotPanic(t, "Noop().Send", func() { n.Send(sampleMessage()) })
}

func TestSendAcceptsSuccessStatuses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv, rec := newNotificationServer(t, status)

			NewWithBaseURL(srv.URL, "").Send(sampleMessage())

			singleRequest(t, rec)
		})
	}
}

func TestSendSurvivesServerError(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv, rec := newNotificationServer(t, status)

			mustNotPanic(t, "Send against a failing server", func() {
				NewWithBaseURL(srv.URL, "internal-secret").Send(sampleMessage())
			})

			singleRequest(t, rec)
		})
	}
}

func TestSendSurvivesUnreachableServer(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	dead.Close()

	n := NewWithBaseURL(dead.URL, "")
	mustNotPanic(t, "Send to a stopped server", func() { n.Send(sampleMessage()) })
}

func TestSendSurvivesInvalidBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"space in host", "http://bad host"},
		{"broken authority", "http://[::1"},
		{"unsupported scheme", "ftp://notifications.internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewWithBaseURL(tt.baseURL, "")
			mustNotPanic(t, "Send with an invalid base URL", func() { n.Send(sampleMessage()) })
		})
	}
}

func TestMessageJSONShape(t *testing.T) {
	withoutReference := sampleMessage()
	withoutReference.Reference = ""

	tests := []struct {
		name     string
		msg      Message
		wantKeys []string
	}{
		{"all fields serialised", sampleMessage(), []string{"userId", "kind", "title", "body", "reference"}},
		{"empty reference omitted", withoutReference, []string{"userId", "kind", "title", "body"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("marshal message: %v", err)
			}

			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatalf("decode message: %v", err)
			}
			if len(payload) != len(tt.wantKeys) {
				t.Errorf("keys = %v, want %v", payload, tt.wantKeys)
			}
			for _, key := range tt.wantKeys {
				if _, ok := payload[key]; !ok {
					t.Errorf("missing key %q in %v", key, payload)
				}
			}
		})
	}
}
