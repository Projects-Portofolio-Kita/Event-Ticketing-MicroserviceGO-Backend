package authz

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	testSecret = "test-secret"
	testToken  = "test-service-token"
)

type guardedResponse struct {
	User     string `json:"user"`
	Internal bool   `json:"internal"`
}

func init() {
	gin.SetMode(gin.TestMode)
}

// sign builds an access token exactly the way auth-service does, unless a
// custom signing method is supplied.
func sign(t *testing.T, secret string, method jwtlib.SigningMethod, claims jwtlib.MapClaims) string {
	t.Helper()

	token, err := jwtlib.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

// defaultClaims is a healthy session for a random user.
func defaultClaims() jwtlib.MapClaims {
	return jwtlib.MapClaims{
		"sub":   uuid.NewString(),
		"email": "user@example.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	}
}

// run mounts the given middleware in front of a probe handler and returns the
// recorder together with the claims the handler observed.
func run(mw gin.HandlerFunc, headers map[string]string) (*httptest.ResponseRecorder, guardedResponse) {
	router := gin.New()
	router.GET("/probe", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, guardedResponse{User: UserID(c), Internal: Internal(c)})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var body guardedResponse
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
	}
	return rec, body
}

func withBearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not a JSON envelope: %v", err)
	}
	if body["message"] == "" {
		t.Fatalf("envelope has no message: %s", rec.Body.String())
	}
}

func TestRequireUserAcceptsValidToken(t *testing.T) {
	claims := defaultClaims()
	rec, body := run(RequireUser(testSecret), withBearer(sign(t, testSecret, jwtlib.SigningMethodHS256, claims)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if body.User != claims["sub"] {
		t.Errorf("user = %q, want %q", body.User, claims["sub"])
	}
	if body.Internal {
		t.Error("a user token must not be reported as an internal call")
	}
}

func TestRequireUserRejectsMissingCredentials(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"no header":    {},
		"empty bearer": {"Authorization": "Bearer "},
		"wrong scheme": {"Authorization": "Basic dXNlcjpwYXNz"},
	} {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, must(run(RequireUser(testSecret), headers)))
		})
	}
}

func TestRequireUserRejectsInvalidTokens(t *testing.T) {
	claims := defaultClaims()

	// A token signed with the wrong secret.
	foreign := sign(t, "another-secret", jwtlib.SigningMethodHS256, claims)

	// An expired token: same secret, exp in the past.
	expiredClaims := defaultClaims()
	expiredClaims["exp"] = time.Now().Add(-time.Minute).Unix()
	expired := sign(t, testSecret, jwtlib.SigningMethodHS256, expiredClaims)

	// A token using an algorithm other than HS256.
	hs512 := sign(t, testSecret, jwtlib.SigningMethodHS512, defaultClaims())

	// A token whose signature has been tampered with.
	valid := sign(t, testSecret, jwtlib.SigningMethodHS256, claims)
	tampered := valid[:len(valid)-2] + swapLastByte(valid)

	// A subject that is not a UUID at all.
	badSubject := sign(t, testSecret, jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"sub": "not-a-uuid",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})

	for name, token := range map[string]string{
		"wrong secret": foreign,
		"expired":      expired,
		"hs512":        hs512,
		"tampered":     tampered,
		"bad subject":  badSubject,
	} {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, must(run(RequireUser(testSecret), withBearer(token))))
		})
	}
}

func TestRequireUserFailsClosedWithoutSecret(t *testing.T) {
	// An unset secret must reject everything, not admit everyone.
	token := sign(t, "", jwtlib.SigningMethodHS256, defaultClaims())
	assertUnauthorized(t, must(run(RequireUser(""), withBearer(token))))
}

func TestRequireServiceAcceptsServiceToken(t *testing.T) {
	rec, body := run(RequireService(testToken), map[string]string{ServiceHeader: testToken})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if !body.Internal {
		t.Error("the service token must be reported as an internal call")
	}
}

func TestRequireServiceRejectsEverythingElse(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"missing":   {},
		"wrong":     {ServiceHeader: "wrong"},
		"user only": withBearer(sign(t, testSecret, jwtlib.SigningMethodHS256, defaultClaims())),
	} {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, must(run(RequireService(testToken), headers)))
		})
	}

	t.Run("unset service token", func(t *testing.T) {
		assertUnauthorized(t, must(run(RequireService(""), map[string]string{ServiceHeader: ""})))
	})
}

func TestRequireUserOrServiceAcceptsBothCredentials(t *testing.T) {
	claims := defaultClaims()

	t.Run("service", func(t *testing.T) {
		rec, body := run(RequireUserOrService(testSecret, testToken), map[string]string{ServiceHeader: testToken})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !body.Internal {
			t.Error("expected an internal caller")
		}
	})

	t.Run("user", func(t *testing.T) {
		rec, body := run(RequireUserOrService(testSecret, testToken), withBearer(sign(t, testSecret, jwtlib.SigningMethodHS256, claims)))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if body.Internal {
			t.Error("expected a user caller")
		}
		if body.User != claims["sub"] {
			t.Errorf("user = %q, want %q", body.User, claims["sub"])
		}
	})
}

func TestRequireUserOrServiceRejectsUnknownCredentials(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"nothing":     {},
		"bad service": {ServiceHeader: "wrong"},
		"bad bearer":  withBearer("garbage"),
		"bad both":    {ServiceHeader: "wrong", "Authorization": "Bearer garbage"},
	} {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, must(run(RequireUserOrService(testSecret, testToken), headers)))
		})
	}
}

func TestWrongServiceTokenStillFallsBackToUserToken(t *testing.T) {
	// A wrong internal header must not poison an otherwise valid user call.
	claims := defaultClaims()
	headers := withBearer(sign(t, testSecret, jwtlib.SigningMethodHS256, claims))
	headers[ServiceHeader] = "wrong"

	rec, body := run(RequireUserOrService(testSecret, testToken), headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if body.Internal || body.User != claims["sub"] {
		t.Fatalf("got %+v, want the user as a non-internal caller", body)
	}
}

func TestParseAccessReturnsSubjectAndEmail(t *testing.T) {
	claims := defaultClaims()
	token := sign(t, testSecret, jwtlib.SigningMethodHS256, claims)

	userID, email, err := ParseAccess(token, testSecret)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if userID != claims["sub"] {
		t.Errorf("userID = %q, want %q", userID, claims["sub"])
	}
	if email != claims["email"] {
		t.Errorf("email = %q, want %q", email, claims["email"])
	}

	if _, _, err := ParseAccess(token, "other-secret"); err == nil {
		t.Error("expected an error for a foreign secret")
	}
	if _, _, err := ParseAccess("", testSecret); err == nil {
		t.Error("expected an error for an empty token")
	}
}

func TestBearerExtraction(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "  Bearer  abc.def.ghi  ")
	if got := bearer(c); got != "abc.def.ghi" {
		t.Errorf("bearer() = %q, want %q", got, "abc.def.ghi")
	}

	c.Request.Header.Set("Authorization", "Token abc")
	if got := bearer(c); got != "" {
		t.Errorf("bearer() = %q for a non-Bearer scheme, want empty", got)
	}

	c.Request.Header.Del("Authorization")
	if got := bearer(c); got != "" {
		t.Errorf("bearer() = %q with no header, want empty", got)
	}
}

// must unwraps the two return values run() produces, keeping call sites short.
func must(rec *httptest.ResponseRecorder, body guardedResponse) *httptest.ResponseRecorder {
	return rec
}

func swapLastByte(token string) string {
	last := token[len(token)-1]
	if last == 'A' {
		return token[:len(token)-1] + "B"
	}
	return token[:len(token)-1] + "A"
}

// Guard against accidental misuse of the header constant.
func TestServiceHeaderName(t *testing.T) {
	if !strings.EqualFold(ServiceHeader, "X-Service-Token") {
		t.Errorf("ServiceHeader = %q, want X-Service-Token", ServiceHeader)
	}
}
