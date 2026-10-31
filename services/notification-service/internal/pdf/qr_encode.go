// Package pdf generates downloadable branded e-ticket PDF passes with QR codes.
//
// File: qr_encode.go
// Usability: Implements QR matrix encoding into PNG bytes and verification payload JSON construction.
package pdf

import (
	"encoding/json"
	"fmt"

	"github.com/skip2/go-qrcode"
)

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
