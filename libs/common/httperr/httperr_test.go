package httperr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// serve runs one GET through the given handlers, where the first handler is
// expected to reply with the error envelope.
func serve(handlers ...gin.HandlerFunc) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET("/resource", handlers...)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/resource", nil))
	return w
}

// helperCase describes one exported helper and the envelope it must write.
type helperCase struct {
	name    string
	call    gin.HandlerFunc
	status  int
	message string
}

// helpers is the full list of exported helpers under test.
func helpers() []helperCase {
	boom := errors.New("upstream exploded")

	return []helperCase{
		{"JSON with a custom status", func(c *gin.Context) { JSON(c, http.StatusForbidden, "akses ditolak") }, http.StatusForbidden, "akses ditolak"},
		{"FromError uses err.Error", func(c *gin.Context) { FromError(c, http.StatusBadGateway, boom) }, http.StatusBadGateway, "upstream exploded"},
		{"BadRequest", func(c *gin.Context) { BadRequest(c, "data tidak valid") }, http.StatusBadRequest, "data tidak valid"},
		{"Unauthorized", func(c *gin.Context) { Unauthorized(c, "silakan masuk") }, http.StatusUnauthorized, "silakan masuk"},
		{"NotFound", func(c *gin.Context) { NotFound(c, "tidak ditemukan") }, http.StatusNotFound, "tidak ditemukan"},
		{"Internal", func(c *gin.Context) { Internal(c, "galat server") }, http.StatusInternalServerError, "galat server"},
	}
}

func TestEnvelopeHoldsExactlyTheMessage(t *testing.T) {
	for _, tt := range helpers() {
		t.Run(tt.name, func(t *testing.T) {
			w := serve(tt.call)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			want := `{"message":"` + tt.message + `"}`
			if got := strings.TrimSpace(w.Body.String()); got != want {
				t.Errorf("body = %s, want %s", got, want)
			}

			var payload map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			if len(payload) != 1 {
				t.Errorf("envelope = %v, want only the message field", payload)
			}
			if msg, _ := payload["message"].(string); msg != tt.message {
				t.Errorf("message = %q, want %q", msg, tt.message)
			}
		})
	}
}

func TestHelpersAbortTheChain(t *testing.T) {
	for _, tt := range helpers() {
		t.Run(tt.name, func(t *testing.T) {
			nextRan := false
			w := serve(tt.call, func(c *gin.Context) {
				nextRan = true
				c.Status(http.StatusOK)
			})

			if nextRan {
				t.Error("the handler after the error helper still ran; the chain was not aborted")
			}
			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
		})
	}
}
