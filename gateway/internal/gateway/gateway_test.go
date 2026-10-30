package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	os.Exit(m.Run())
}

// gatewayClient bounds how long any single gateway call may take.
var gatewayClient = &http.Client{Timeout: 10 * time.Second}

// startGateway serves the engine the way production does: gin behind a real
// net/http server. Driving gin in-process with httptest.ResponseRecorder does
// not work for the proxying routes: gin's ResponseWriter claims to implement
// http.CloseNotifier while the recorder does not, and httputil.ReverseProxy
// consults CloseNotifier whenever the request context carries no cancellation
// signal (which is exactly what httptest.NewRequest produces).
func startGateway(t *testing.T, g *Gateway) string {
	t.Helper()
	srv := httptest.NewServer(g.Engine())
	t.Cleanup(srv.Close)
	return srv.URL
}

// gatewaySend executes req against a started gateway and returns the response
// with its body fully read and closed.
func gatewaySend(t *testing.T, req *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := gatewayClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("%s %s: read body: %v", req.Method, req.URL, err)
	}
	return resp, string(raw)
}

// gatewayGet is gatewaySend for a plain GET.
func gatewayGet(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	return gatewaySend(t, req)
}

// isolateUpstreams pins every routing row to base, so New() never depends on
// the ambient environment.
func isolateUpstreams(t *testing.T, base string) {
	t.Helper()
	for _, up := range Upstreams {
		t.Setenv(up.EnvKey, base)
	}
}

// upstreamRequest is one request captured by a fake backend.
type upstreamRequest struct {
	method   string
	path     string
	rawQuery string
	header   http.Header
	body     []byte
}

// upstreamRecorder collects requests without racing the handler goroutine.
type upstreamRecorder struct {
	mu   sync.Mutex
	reqs []upstreamRequest
}

func (r *upstreamRecorder) add(req upstreamRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
}

func (r *upstreamRecorder) all() []upstreamRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]upstreamRequest(nil), r.reqs...)
}

// newUpstream starts a backend that records every request and then delegates
// to next.
func newUpstream(t *testing.T, next http.HandlerFunc) (*httptest.Server, *upstreamRecorder) {
	t.Helper()
	rec := &upstreamRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.add(upstreamRequest{
			method:   r.Method,
			path:     r.URL.Path,
			rawQuery: r.URL.RawQuery,
			header:   r.Header.Clone(),
			body:     body,
		})
		next(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// proxiedJSON answers every path with a small JSON document.
func proxiedJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"proxied":true}`))
}

// healthyService mimics a backend that publishes GET /health.
func healthyService(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"catalog-service"}`))
}

// unhealthyService is reachable but reports itself as degraded.
func unhealthyService(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"status":"degraded","service":"catalog-service"}`))
}

// stoppedUpstream returns the URL of an httptest server that is already shut
// down, so connections to it fail immediately.
func stoppedUpstream(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()
	return srv.URL
}

func TestRewritePath(t *testing.T) {
	auth := Upstream{APIFrom: "/api/auth", APITo: "/api/v1/auth"}
	catalog := Upstream{APIFrom: "/api/catalog", APITo: "/api/v1"}
	notification := Upstream{APIFrom: "/api/notifications", APITo: "/api/v1/notifications"}

	tests := []struct {
		name string
		up   Upstream
		path string
		want string
	}{
		{"exact public path maps to the service root", auth, "/api/auth", "/api/v1/auth"},
		{"sub path keeps its suffix", auth, "/api/auth/login", "/api/v1/auth/login"},
		{"catalog prefix collapses onto /api/v1", catalog, "/api/catalog/events", "/api/v1/events"},
		{"longest prefix is replaced as a whole", notification, "/api/notifications/42/read", "/api/v1/notifications/42/read"},
		{"exact notifications collection", notification, "/api/notifications", "/api/v1/notifications"},
		{"trailing slash is normalised", notification, "/api/notifications/", "/api/v1/notifications"},
		// rewritePath never checks APIFrom: routing guarantees the match, so a
		// foreign path still gets APITo prepended instead of being left intact.
		{"path outside the prefix is not left intact", auth, "/health", "/api/v1/auth/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rewritePath(tt.up, tt.path); got != tt.want {
				t.Errorf("rewritePath(%q, %q) = %q, want %q", tt.up.APIFrom, tt.path, got, tt.want)
			}
		})
	}
}

func TestUpstreamTargetReadsEnvOverride(t *testing.T) {
	up := Upstream{
		Name:       "auth",
		Service:    "auth-service",
		EnvKey:     "AUTH_SERVICE_URL",
		DefaultURL: "http://auth-service:8081",
	}

	t.Run("env wins over the default", func(t *testing.T) {
		t.Setenv(up.EnvKey, "http://auth-override:9090/")
		if got, want := up.target(), "http://auth-override:9090"; got != want {
			t.Errorf("target() = %q, want %q (trailing slash trimmed)", got, want)
		}
	})

	t.Run("env without trailing slash is kept", func(t *testing.T) {
		t.Setenv(up.EnvKey, "http://127.0.0.1:8081")
		if got, want := up.target(), "http://127.0.0.1:8081"; got != want {
			t.Errorf("target() = %q, want %q", got, want)
		}
	})

	t.Run("empty env falls back to the default", func(t *testing.T) {
		t.Setenv(up.EnvKey, "")
		if got, want := up.target(), up.DefaultURL; got != want {
			t.Errorf("target() = %q, want %q", got, want)
		}
	})
}

// TestForwardRewritesPathAndStripsServiceToken also proves New() honours the
// *_SERVICE_URL override: every upstream points at the recording backend.
func TestForwardRewritesPathAndStripsServiceToken(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		body             string
		wantQuery        string
		wantUpstreamPath string
	}{
		{"prefix is rewritten", http.MethodGet, "/api/catalog/events?limit=10", "", "limit=10", "/api/v1/events"},
		{"exact prefix path", http.MethodGet, "/api/catalog", "", "", "/api/v1"},
		{"auth prefix with body", http.MethodPost, "/api/auth/login", `{"email":"andi@example.com"}`, "", "/api/v1/auth/login"},
		{"notifications sub path", http.MethodGet, "/api/notifications/42/read", "", "", "/api/v1/notifications/42/read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rec := newUpstream(t, proxiedJSON)
			isolateUpstreams(t, srv.URL)
			base := startGateway(t, New())

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req, err := http.NewRequest(tt.method, base+tt.path, body)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Service-Token", "must-not-reach-upstream")
			req.Header.Set("Origin", "https://portal.example.com")
			resp, raw := gatewaySend(t, req)

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d (body %q)", resp.StatusCode, http.StatusOK, raw)
			}

			reqs := rec.all()
			if len(reqs) != 1 {
				t.Fatalf("upstream captured %d requests, want 1", len(reqs))
			}
			got := reqs[0]
			if got.path != tt.wantUpstreamPath {
				t.Errorf("upstream path = %q, want %q", got.path, tt.wantUpstreamPath)
			}
			if got.rawQuery != tt.wantQuery {
				t.Errorf("upstream query = %q, want %q", got.rawQuery, tt.wantQuery)
			}
			if string(got.body) != tt.body {
				t.Errorf("upstream body = %q, want %q", got.body, tt.body)
			}
			if _, present := got.header["X-Service-Token"]; present {
				t.Error("X-Service-Token from the caller reached the upstream; it must be stripped")
			}

			if relayed := strings.TrimSpace(raw); relayed != `{"proxied":true}` {
				t.Errorf("relayed body = %s, want %s", relayed, `{"proxied":true}`)
			}
			if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "*" {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "*")
			}
		})
	}
}

func TestNoRouteRepliesJSONNotFound(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"unknown path", "/definitely/not/routed"},
		{"near miss of a public prefix", "/api/notificationsx"},
		{"bare api root", "/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rec := newUpstream(t, proxiedJSON)
			isolateUpstreams(t, srv.URL)
			base := startGateway(t, New())

			resp, raw := gatewayGet(t, base+tt.path)

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Content-Type = %q, want application/json (not plain text)", ct)
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				t.Fatalf("decode 404 body %q: %v", raw, err)
			}
			if len(payload) != 1 {
				t.Errorf("body = %v, want only the message field", payload)
			}
			if msg, _ := payload["message"].(string); msg == "" {
				t.Error("message field is missing or empty")
			}
			if n := len(rec.all()); n != 0 {
				t.Errorf("unrouted path reached an upstream %d time(s)", n)
			}
		})
	}
}

func TestHealthProxyForSingleService(t *testing.T) {
	t.Run("healthy upstream is relayed", func(t *testing.T) {
		srv, rec := newUpstream(t, healthyService)
		isolateUpstreams(t, srv.URL)
		base := startGateway(t, New())

		resp, raw := gatewayGet(t, base+"/health/catalog")

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		want := `{"status":"ok","service":"catalog-service"}`
		if got := strings.TrimSpace(raw); got != want {
			t.Errorf("body = %s, want %s", got, want)
		}
		reqs := rec.all()
		if len(reqs) != 1 {
			t.Fatalf("upstream captured %d requests, want 1", len(reqs))
		}
		if reqs[0].path != "/health" {
			t.Errorf("upstream path = %q, want %q", reqs[0].path, "/health")
		}
	})

	t.Run("unhealthy upstream status is relayed", func(t *testing.T) {
		srv, _ := newUpstream(t, unhealthyService)
		isolateUpstreams(t, srv.URL)
		base := startGateway(t, New())

		resp, raw := gatewayGet(t, base+"/health/catalog")

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
		}
		want := `{"status":"degraded","service":"catalog-service"}`
		if got := strings.TrimSpace(raw); got != want {
			t.Errorf("body = %s, want %s", got, want)
		}
	})

	t.Run("stopped upstream reports down", func(t *testing.T) {
		dead := stoppedUpstream(t)
		isolateUpstreams(t, dead)
		base := startGateway(t, New())

		resp, raw := gatewayGet(t, base+"/health/catalog")

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
		}
		var payload map[string]string
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode body %q: %v", raw, err)
		}
		if payload["status"] != "down" {
			t.Errorf("status = %q, want %q", payload["status"], "down")
		}
		if want := strings.TrimPrefix(dead, "http://"); payload["service"] != want {
			t.Errorf("service = %q, want %q", payload["service"], want)
		}
	})
}

func TestAggregateHealthReportsAllServicesUp(t *testing.T) {
	srv, rec := newUpstream(t, healthyService)
	isolateUpstreams(t, srv.URL)
	base := startGateway(t, New())

	resp, raw := gatewayGet(t, base+"/health")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var payload struct {
		Status   string            `json:"status"`
		Gateway  string            `json:"gateway"`
		Services map[string]string `json:"services"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode aggregate health: %v", err)
	}
	if payload.Status != "ok" {
		t.Errorf("status = %q, want %q", payload.Status, "ok")
	}
	if payload.Gateway != "gateway" {
		t.Errorf("gateway = %q, want %q", payload.Gateway, "gateway")
	}
	if len(payload.Services) != len(Upstreams) {
		t.Errorf("services = %v, want %d entries", payload.Services, len(Upstreams))
	}
	for _, up := range Upstreams {
		if payload.Services[up.Name] != "up" {
			t.Errorf("services[%q] = %q, want %q", up.Name, payload.Services[up.Name], "up")
		}
	}

	reqs := rec.all()
	if len(reqs) != len(Upstreams) {
		t.Fatalf("probed %d backends, want %d", len(reqs), len(Upstreams))
	}
	for _, req := range reqs {
		if req.path != "/health" {
			t.Errorf("probe path = %q, want %q", req.path, "/health")
		}
	}
}

func TestAggregateHealthDegradedWhenOneUpstreamIsDown(t *testing.T) {
	srv, _ := newUpstream(t, healthyService)
	isolateUpstreams(t, srv.URL)
	t.Setenv("PAYMENT_SERVICE_URL", stoppedUpstream(t))
	base := startGateway(t, New())

	resp, raw := gatewayGet(t, base+"/health")

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	var payload struct {
		Status   string            `json:"status"`
		Services map[string]string `json:"services"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode aggregate health: %v", err)
	}
	if payload.Status != "degraded" {
		t.Errorf("status = %q, want %q", payload.Status, "degraded")
	}
	if payload.Services["payment"] != "down" {
		t.Errorf("services[\"payment\"] = %q, want %q", payload.Services["payment"], "down")
	}
	for _, up := range Upstreams {
		if up.Name == "payment" {
			continue
		}
		if payload.Services[up.Name] != "up" {
			t.Errorf("services[%q] = %q, want %q", up.Name, payload.Services[up.Name], "up")
		}
	}
}

func TestForwardToUnreachableUpstreamRepliesBadGateway(t *testing.T) {
	isolateUpstreams(t, stoppedUpstream(t))
	base := startGateway(t, New())

	req, err := http.NewRequest(http.MethodPost, base+"/api/booking/seats", strings.NewReader(`{"seat":"A1"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, raw := gatewaySend(t, req)

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode body %q: %v", raw, err)
	}
	if !strings.Contains(payload["message"], "booking-service") {
		t.Errorf("message = %q, want it to name booking-service", payload["message"])
	}
}
