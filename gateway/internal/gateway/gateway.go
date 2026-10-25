// Package gateway is the single entry point in front of every service.
//
// It exposes one stable public surface:
//
//	/api/auth/*          -> auth-service         :8081  (/api/auth    -> /api/v1/auth)
//	/api/catalog/*       -> catalog-service      :8082  (/api/catalog -> /api/v1)
//	/api/booking/*       -> booking-service      :8083  (/api/booking -> /api/v1)
//	/api/payment/*       -> payment-service      :8084  (/api/payment -> /api/v1)
//	/api/notifications/* -> notification-service :8085  (/api/notifications -> /api/v1)
//	/health/<name>       -> that service's /health
//	/health              -> aggregated view of all of the above
//
// The rewrite rules are the same ones implemented by the Vite dev proxy and
// the portal's nginx.conf, so a client can switch between "direct to gateway"
// and "through the portal" without changing its request paths.
package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/gin-gonic/gin"
)

// Upstream is one backend service routed by the gateway.
type Upstream struct {
	// Name is the short key used in health output, e.g. "auth".
	Name string
	// Service is the compose service name shown to operators, e.g. "auth-service".
	Service string
	// EnvKey overrides the target URL, e.g. "AUTH_SERVICE_URL".
	EnvKey string
	// DefaultURL is used when EnvKey is unset.
	DefaultURL string
	// APIFrom is the public path prefix, e.g. "/api/auth".
	APIFrom string
	// APITo replaces APIFrom before forwarding, e.g. "/api/v1/auth".
	APITo string
}

// Upstreams is the routing table. Adding a service means adding a row here
// plus a compose entry — no other change.
var Upstreams = []Upstream{
	{Name: "auth", Service: "auth-service", EnvKey: "AUTH_SERVICE_URL", DefaultURL: "http://auth-service:8081", APIFrom: "/api/auth", APITo: "/api/v1/auth"},
	{Name: "catalog", Service: "catalog-service", EnvKey: "CATALOG_SERVICE_URL", DefaultURL: "http://catalog-service:8082", APIFrom: "/api/catalog", APITo: "/api/v1"},
	{Name: "booking", Service: "booking-service", EnvKey: "BOOKING_SERVICE_URL", DefaultURL: "http://booking-service:8083", APIFrom: "/api/booking", APITo: "/api/v1"},
	{Name: "payment", Service: "payment-service", EnvKey: "PAYMENT_SERVICE_URL", DefaultURL: "http://payment-service:8084", APIFrom: "/api/payment", APITo: "/api/v1"},
	{Name: "notification", Service: "notification-service", EnvKey: "NOTIFICATION_SERVICE_URL", DefaultURL: "http://notification-service:8085", APIFrom: "/api/notifications", APITo: "/api/v1/notifications"},
}

// target resolves the upstream URL for a row (env var wins).
func (u Upstream) target() string {
	if v := strings.TrimRight(os.Getenv(u.EnvKey), "/"); v != "" {
		return v
	}
	return u.DefaultURL
}

// Gateway holds the router and the resolved upstream targets.
type Gateway struct {
	engine  *gin.Engine
	targets map[string]*url.URL
}

// New builds the gateway router.
func New() *Gateway {
	g := &Gateway{
		engine:  gin.New(),
		targets: make(map[string]*url.URL, len(Upstreams)),
	}

	g.engine.Use(gin.Logger(), gin.Recovery(), cors.New())

	// A gateway must never answer with a redirect: clients (and the docs)
	// treat every path as opaque and forward it as-is.
	g.engine.RedirectTrailingSlash = false
	g.engine.RedirectFixedPath = false

	for _, up := range Upstreams {
		target, err := url.Parse(up.target())
		if err != nil {
			// A malformed URL would break one route only; fail loudly instead
			// of silently proxying to a nonsense address.
			panic("gateway: invalid URL for " + up.Service + ": " + err.Error())
		}
		g.targets[up.Name] = target

		handler := g.forward(up, target)
		// Exact path (e.g. GET /api/notifications?userId=) *and* everything
		// below it: without the exact route gin would 301 to a trailing slash.
		g.engine.Any(up.APIFrom, handler)
		g.engine.Any(up.APIFrom+"/*rest", handler)
		g.engine.GET("/health/"+up.Name, g.healthOf(target))
	}

	g.engine.GET("/health", g.aggregateHealth)
	g.engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "route tidak ditemukan di gateway"})
	})

	return g
}

// Engine returns the configured router.
func (g *Gateway) Engine() *gin.Engine { return g.engine }

// forward proxies a public prefix to its service with the prefix rewritten.
func (g *Gateway) forward(up Upstream, target *url.URL) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(gin.H{"message": up.Service + " tidak dapat dihubungi"})
	}

	return func(c *gin.Context) {
		req := c.Request
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = rewritePath(up, req.URL.Path)
		req.Host = target.Host

		// The internal credential must never be accepted from the public
		// internet: strip whatever the caller sent so upstream services can
		// only be reached with a token that originated inside the network.
		req.Header.Del(authz.ServiceHeader)

		// httputil reads the body itself; mark gin's copy as consumed.
		proxy.ServeHTTP(c.Writer, req)
		c.Abort()
	}
}

// rewritePath maps a public path onto its upstream path.
//
//	/api/auth/login            -> /api/v1/auth/login
//	/api/catalog/events        -> /api/v1/events
//	/api/notifications         -> /api/v1/notifications   (collection at the prefix)
//	/api/notifications/        -> /api/v1/notifications   (trailing slash normalized)
//	/api/notifications/{id}/read -> /api/v1/notifications/{id}/read
func rewritePath(up Upstream, path string) string {
	rest := strings.TrimSuffix(strings.TrimPrefix(path, up.APIFrom), "/")
	if rest == "" {
		return up.APITo
	}
	return up.APITo + rest
}

// healthOf proxies GET /health/<name> to the service's own /health.
func (g *Gateway) healthOf(target *url.URL) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := http.Get(target.Scheme + "://" + target.Host + "/health")
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "service": target.Host})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "service": target.Host})
			return
		}

		c.Data(resp.StatusCode, "application/json", body)
	}
}

// aggregateHealth reports every upstream in one call, so a client (or the
// landing page) can show the whole platform's status with a single request.
func (g *Gateway) aggregateHealth(c *gin.Context) {
	services := make(map[string]string, len(Upstreams))
	allUp := true

	for _, up := range Upstreams {
		status, code := probe(up.target())
		services[up.Name] = status
		if code != http.StatusOK {
			allUp = false
		}
	}

	status := "ok"
	if !allUp {
		status = "degraded"
	}
	payload := gin.H{"status": status, "gateway": "gateway", "services": services}
	if status != "ok" {
		c.JSON(http.StatusServiceUnavailable, payload)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func probe(target string) (string, int) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(target, "/") + "/health")
	if err != nil {
		return "down", http.StatusServiceUnavailable
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusOK {
		return "down", resp.StatusCode
	}
	return "up", http.StatusOK
}
