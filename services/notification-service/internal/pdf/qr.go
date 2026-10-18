package pdf

import (
	"encoding/json"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// DefaultQRSize is the default resolution (width and height in pixels) for generated QR codes.
const DefaultQRSize = 256

// VerificationPayload represents the standardized verification data encoded in the QR code.
type VerificationPayload struct {
	BookingRef string `json:"bookingRef"`
	SeatID     string `json:"seatId"`
	EventID    string `json:"eventId"`
}

// QRCodeGenerator defines the contract for QR code creation and verification payload encoding.
type QRCodeGenerator interface {
	Generate(payload string) ([]byte, error)
	GenerateVerificationPayload(bookingRef, seatID, eventID string) string
}

// QRGenerator implements QRCodeGenerator using skip2/go-qrcode.
type QRGenerator struct {
	size     int
	recovery qrcode.RecoveryLevel
}

// NewQRCodeGenerator creates a new QRGenerator with the specified size and High recovery level.
// If size <= 0, DefaultQRSize (256) is used.
func NewQRCodeGenerator(size int) *QRGenerator {
	if size <= 0 {
		size = DefaultQRSize
	}
	return &QRGenerator{
		size:     size,
		recovery: qrcode.High,
	}
}

// Generate creates a high-resolution PNG QR code from the given payload string.
func (q *QRGenerator) Generate(payload string) ([]byte, error) {
	if payload == "" {
		return nil, fmt.Errorf("payload cannot be empty")
	}
	pngBytes, err := qrcode.Encode(payload, q.recovery, q.size)
	if err != nil {
		return nil, fmt.Errorf("failed to encode qr code: %w", err)
	}
	return pngBytes, nil
}

// GenerateVerificationPayload builds a standardized JSON verification payload.
func (q *QRGenerator) GenerateVerificationPayload(bookingRef, seatID, eventID string) string {
	payload := VerificationPayload{
		BookingRef: bookingRef,
		SeatID:     seatID,
		EventID:    eventID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"bookingRef":"%s","seatId":"%s","eventId":"%s"}`, bookingRef, seatID, eventID)
	}
	return string(bytes)
}
