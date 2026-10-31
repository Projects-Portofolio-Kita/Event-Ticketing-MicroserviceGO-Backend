// Package pdf generates downloadable branded e-ticket PDF passes with QR codes.
//
// File: qr.go
// Usability: Declares VerificationPayload model, QRCodeGenerator interface, and QRGenerator struct.
package pdf

import "github.com/skip2/go-qrcode"

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
func NewQRCodeGenerator(size int) *QRGenerator {
	if size <= 0 {
		size = DefaultQRSize
	}
	return &QRGenerator{
		size:     size,
		recovery: qrcode.High,
	}
}
