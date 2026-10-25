// Package authz is the shared authentication/authorization layer used by the
// services that own user data (booking, payment, notification).
//
// Two credentials exist:
//
//   - user — the HS256 access JWT issued by auth-service
//     (Authorization: Bearer <token>, claims: sub, email, exp). It proves
//     *who* the caller is, so handlers enforce ownership with UserID().
//
//   - service — the X-Service-Token header, a shared secret proving the call
//     comes from another service inside the compose network. It proves *what*
//     is calling, not who, so it is used for service-to-service endpoints
//     (payment -> booking, booking -> notification). The gateway strips this
//     header from incoming public traffic, so it never reaches the internet.
//
// Both credentials fail closed: an empty secret/token rejects everything
// instead of silently allowing it.
package authz

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ServiceHeader carries the internal service-to-service credential.
const ServiceHeader = "X-Service-Token"

// Context keys set on the gin context by the middleware below.
const (
	userIDKey   = "authz.userID"
	emailKey    = "authz.email"
	internalKey = "authz.internal"
)

type mode int

const (
	modeUser    mode = iota // only a user JWT
	modeService             // only the service token
	modeEither              // service token (trusted) or user JWT
)

// RequireUser accepts only a valid user access JWT.
// The subject is then available through UserID(c).
func RequireUser(secret string) gin.HandlerFunc {
	return guard(secret, "", modeUser)
}

// RequireService accepts only the X-Service-Token header. Use it for endpoints
// that may only be called by another service (no user is involved).
func RequireService(serviceToken string) gin.HandlerFunc {
	return guard("", serviceToken, modeService)
}

// RequireUserOrService accepts either credential: a user JWT, or the service
// token for internal callers. Use Internal(c) to skip ownership checks for
// service-to-service traffic.
func RequireUserOrService(secret, serviceToken string) gin.HandlerFunc {
	return guard(secret, serviceToken, modeEither)
}

func guard(secret, serviceToken string, m mode) gin.HandlerFunc {
	if m != modeService && secret == "" {
		log.Println("authz: JWT_SECRET kosong — permintaan bertanda Bearer akan selalu ditolak")
	}
	if m != modeUser && serviceToken == "" {
		log.Println("authz: SERVICE_TOKEN kosong — panggilan antar-service akan selalu ditolak")
	}

	return func(c *gin.Context) {
		if m != modeUser && serviceToken != "" {
			got := c.GetHeader(ServiceHeader)
			if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(serviceToken)) == 1 {
				c.Set(internalKey, true)
				c.Next()
				return
			}
		}

		if m == modeService {
			abort(c, http.StatusUnauthorized, "Token layanan tidak valid.")
			return
		}

		raw := bearer(c)
		if raw == "" {
			abort(c, http.StatusUnauthorized, "Token tidak ditemukan. Silakan masuk.")
			return
		}

		userID, email, err := parseAccess(raw, secret)
		if err != nil {
			abort(c, http.StatusUnauthorized, "Sesi kedaluwarsa. Silakan masuk ulang.")
			return
		}

		c.Set(userIDKey, userID)
		c.Set(emailKey, email)
		c.Set(internalKey, false)
		c.Next()
	}
}

// Caller is the authenticated principal of a request: the user whose access
// JWT was verified, or an internal service call authenticated with the
// service token (which owns nothing, so ownership checks are skipped).
type Caller struct {
	UserID   string
	Internal bool
}

// CallerFrom builds the Caller that the middleware stored on the context.
func CallerFrom(c *gin.Context) Caller {
	return Caller{UserID: UserID(c), Internal: Internal(c)}
}

// UserID returns the authenticated subject, or "" when there is none.
func UserID(c *gin.Context) string {
	v, _ := c.Get(userIDKey)
	s, _ := v.(string)
	return s
}

// Email returns the e-mail claim of the authenticated subject, or "".
func Email(c *gin.Context) string {
	v, _ := c.Get(emailKey)
	s, _ := v.(string)
	return s
}

// Internal reports whether the request was authenticated with the service
// token instead of a user JWT.
func Internal(c *gin.Context) bool {
	v, _ := c.Get(internalKey)
	b, _ := v.(bool)
	return b
}

// abort replies with the standard {"message": "..."} envelope used by every
// service, then stops the request chain.
func abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"message": message})
}

// bearer extracts the raw token from the Authorization header.
func bearer(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

// parseAccess validates an access JWT exactly the way auth-service signs it:
// HS256, sub must be a UUID, exp checked by the library.
func parseAccess(raw, secret string) (userID, email string, err error) {
	if secret == "" {
		return "", "", errors.New("jwt secret not configured")
	}

	token, err := jwtlib.Parse(raw, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwtlib.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", "", errors.New("token missing subject")
	}
	if _, err := uuid.Parse(sub); err != nil {
		return "", "", errors.New("token subject is not a uuid")
	}

	email, _ = claims["email"].(string)
	return sub, email, nil
}

// ParseAccess exposes the validation used by the middleware, for handlers that
// need the subject without running the middleware (e.g. tests).
func ParseAccess(raw, secret string) (userID, email string, err error) {
	return parseAccess(raw, secret)
}

// RequireEnv reads a mandatory environment variable for a service that cannot
// start without it. Intended for main(): a service missing its credentials
// must fail loudly instead of serving unauthenticated traffic.
func RequireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("authz: %s wajib diisi (contoh: openssl rand -hex 32)", key)
	}
	return v
}
