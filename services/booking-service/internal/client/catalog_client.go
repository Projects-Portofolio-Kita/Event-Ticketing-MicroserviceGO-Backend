package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
)

// ErrEventNotFound is returned when catalog-service reports the event does not
// exist, so the service can answer 404 instead of a generic 500.
var ErrEventNotFound = errors.New("event tidak ditemukan")

// TierQuota is one tier of an event as catalog-service knows it: name, price
// and total stock. Prices come from here on purpose — a client can never say
// what a ticket costs.
type TierQuota struct {
	CategoryID string  `json:"categoryId"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Quota      int     `json:"quota"`
}

// CatalogClient is the slice of catalog-service booking-service depends on.
// Tests substitute an in-memory fake.
type CatalogClient interface {
	// GetTierQuotas returns every tier (price + quota) of an event.
	GetTierQuotas(eventID string) ([]TierQuota, error)
}

type HTTPCatalogClient struct {
	baseURL      string
	serviceToken string
	http         *http.Client
}

func NewHTTPCatalogClient() *HTTPCatalogClient {
	baseURL := os.Getenv("CATALOG_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://catalog-service:8082"
	}

	return &HTTPCatalogClient{
		baseURL:      baseURL,
		serviceToken: os.Getenv("SERVICE_TOKEN"),
		http:         &http.Client{Timeout: 3 * time.Second},
	}
}

// GetTierQuotas reads catalog-service's internal stock feed. The feed is only
// reachable with X-Service-Token: without it every tier booking would fail.
func (c *HTTPCatalogClient) GetTierQuotas(eventID string) ([]TierQuota, error) {
	url := fmt.Sprintf("%s/api/v1/events/%s/tier-quotas", c.baseURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.serviceToken != "" {
		req.Header.Set(authz.ServiceHeader, c.serviceToken)
	}

	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, ErrEventNotFound
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog-service returned status %d", response.StatusCode)
	}

	var body struct {
		Tiers []TierQuota `json:"tiers"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode tier quotas: %w", err)
	}

	return body.Tiers, nil
}
