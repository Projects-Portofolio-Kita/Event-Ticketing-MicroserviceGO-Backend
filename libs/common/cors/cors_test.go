package cors

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	allowOrigin   = "*"
	allowMethods  = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowHeaders  = "Origin, Content-Type, Accept, Authorization"
	probePath     = "/resource"
	unknownOrigin = "http://evil.example"
)

// newTestEngine wires the middleware under test in front of a probe handler
// that echoes method and body, so a broken chain is visible in the response.
func newTestEngine() (*gin.Engine, func() bool) {
	reached := false
	r := gin.New()
	r.Use(New())
	r.Any(probePath, func(c *gin.Context) {
		reached = true
		b, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, c.Request.Method+":"+string(b))
	})
	return r, func() bool { return reached }
}

// The policy is a wildcard, so every origin is answered identically — there
// is no allow-list and no origin is ever rejected.
var corsTests = []struct {
	name   string
	origin string
}{
	{"dev portal origin", "http://localhost:5173"},
	{"portal origin", "https://tickets.example.com"},
	{"unknown origin", unknownOrigin},
}

func TestPreflightIsAnsweredAndAbortsTheChain(t *testing.T) {
	for _, tt := range corsTests {
		t.Run(tt.name, func(t *testing.T) {
			r, reached := newTestEngine()

			req := httptest.NewRequest(http.MethodOptions, probePath, nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNoContent {
				t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != allowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowOrigin)
			}
			if got := w.Header().Get("Access-Control-Allow-Methods"); got != allowMethods {
				t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, allowMethods)
			}
			if got := w.Header().Get("Access-Control-Allow-Headers"); got != allowHeaders {
				t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, allowHeaders)
			}
			if w.Body.Len() != 0 {
				t.Errorf("preflight body = %q, want empty", w.Body.String())
			}
			if reached() {
				t.Error("preflight reached the handler; the chain must be aborted")
			}
		})
	}
}

func TestSimpleRequestReachesHandlerWithCORSHeaders(t *testing.T) {
	for _, tt := range corsTests {
		t.Run(tt.name, func(t *testing.T) {
			r, reached := newTestEngine()

			req := httptest.NewRequest(http.MethodGet, probePath, nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if got, want := w.Body.String(), "GET:"; got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
			if !reached() {
				t.Error("handler was not reached; CORS must not stop normal requests")
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != allowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowOrigin)
			}
		})
	}
}

func TestOtherMethodsPassThroughUntouched(t *testing.T) {
	tests := []struct {
		method string
		body   string
	}{
		{http.MethodGet, ""},
		{http.MethodPost, `{"seat":"A1"}`},
		{http.MethodPut, `{"seat":"A2"}`},
		{http.MethodPatch, `{"name":"andi"}`},
		{http.MethodDelete, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			r, reached := newTestEngine()

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tt.method, probePath, body))

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if got, want := w.Body.String(), tt.method+":"+tt.body; got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
			if !reached() {
				t.Error("handler was not reached")
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != allowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowOrigin)
			}
		})
	}
}
