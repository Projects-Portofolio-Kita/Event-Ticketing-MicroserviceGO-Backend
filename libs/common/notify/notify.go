// Package notify is the shared, fire-and-forget client for notification-service.
//
// Booking- and payment-service both emit user-facing events (seat locked,
// booking confirmed, payment succeeded). Instead of each service keeping its
// own HTTP client, they call this one. A notification is never allowed to
// break the business flow: failures are logged and swallowed.
//
// Configuration: NOTIFICATION_SERVICE_URL (e.g. http://notification-service:8085).
// When unset, Send is a no-op, so services still work without the service.
package notify

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// Kinds emitted by the services. Keep in sync with contracts/events.
const (
	KindReservationLocked = "reservation.locked"
	KindBookingConfirmed  = "booking.confirmed"
	KindBookingCancelled  = "booking.cancelled"
	KindPaymentSucceeded  = "payment.succeeded"
)

// Message is the payload posted to POST {base}/api/v1/notifications.
type Message struct {
	UserID    string `json:"userId"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Reference string `json:"reference,omitempty"`
}

// Notifier delivers messages to notification-service.
type Notifier interface {
	Send(Message)
}

// New reads NOTIFICATION_SERVICE_URL and returns a Notifier.
// An unset URL yields a no-op Notifier.
func New() Notifier {
	return NewWithBaseURL(os.Getenv("NOTIFICATION_SERVICE_URL"))
}

// Noop returns a Notifier that discards every message. Useful as a fallback
// when a caller was not given one.
func Noop() Notifier { return noop{} }

// NewWithBaseURL is New with an explicit base URL; empty URL disables sending.
func NewWithBaseURL(baseURL string) Notifier {
	if baseURL == "" {
		return noop{}
	}
	return &httpNotifier{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

type noop struct{}

func (noop) Send(Message) {}

type httpNotifier struct {
	baseURL string
	client  *http.Client
}

func (n *httpNotifier) Send(msg Message) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("notify: marshal %s: %v", msg.Kind, err)
		return
	}

	resp, err := n.client.Post(
		n.baseURL+"/api/v1/notifications",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		log.Printf("notify: %s failed: %v", msg.Kind, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("notify: %s rejected with status %d", msg.Kind, resp.StatusCode)
	}
}
