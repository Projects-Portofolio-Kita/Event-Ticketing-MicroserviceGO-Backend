package pdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// PNGMagicHeader represents the standard 8-byte signature of a valid PNG file
var PNGMagicHeader = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// PDFMagicHeader represents the binary signature of a valid PDF document
var PDFMagicHeader = []byte("%PDF-")

func TestQRCode_Generation(t *testing.T) {
	qrGen := NewQRCodeGenerator(256)

	payload := qrGen.GenerateVerificationPayload("TICKID-REF-9988", "SEAT-VIP-A1", "EVT-CONCERT-001")
	if !strings.Contains(payload, "TICKID-REF-9988") {
		t.Fatalf("expected payload to contain booking ref, got %s", payload)
	}

	qrBytes, err := qrGen.Generate(payload)
	if err != nil {
		t.Fatalf("unexpected error generating QR code: %v", err)
	}

	if len(qrBytes) == 0 {
		t.Fatal("expected non-empty byte slice for QR code PNG")
	}

	// Validate PNG magic bytes header \x89PNG\r\n\x1a\n
	if !bytes.HasPrefix(qrBytes, PNGMagicHeader) {
		t.Fatalf("generated QR bytes do not start with PNG magic header, got prefix: %x", qrBytes[:8])
	}

	// Check minimum reasonable PNG size (> 500 bytes for 256x256 QR)
	if len(qrBytes) < 500 {
		t.Fatalf("generated QR code size unexpectedly small: %d bytes", len(qrBytes))
	}
}

func TestQRCode_EmptyPayload(t *testing.T) {
	qrGen := NewQRCodeGenerator(256)
	_, err := qrGen.Generate("")
	if err == nil {
		t.Fatal("expected error when generating QR code with empty payload, got nil")
	}
}

func TestQRCode_CustomSize(t *testing.T) {
	// Custom size 512
	gen512 := NewQRCodeGenerator(512)
	bytes512, err := gen512.Generate("TICKID-HIGHRES")
	if err != nil {
		t.Fatalf("failed to generate 512x512 QR code: %v", err)
	}
	if !bytes.HasPrefix(bytes512, PNGMagicHeader) {
		t.Fatal("512x512 QR does not have valid PNG header")
	}

	// Zero / negative size falls back to DefaultQRSize (256)
	genDefault := NewQRCodeGenerator(-1)
	bytesDefault, err := genDefault.Generate("TICKID-DEFAULT")
	if err != nil {
		t.Fatalf("failed to generate default QR code: %v", err)
	}
	if !bytes.HasPrefix(bytesDefault, PNGMagicHeader) {
		t.Fatal("default QR does not have valid PNG header")
	}
}

func TestQRCode_MultiSeatUniqueness(t *testing.T) {
	qrGen := NewQRCodeGenerator(256)

	eventID := "EVT-STADIUM-ROCK"
	bookingRef := "BOOK-MULTI-1234"

	// Generate QR for Seat 1
	payloadSeat1 := qrGen.GenerateVerificationPayload(bookingRef, "A-01", eventID)
	qrBytesSeat1, err := qrGen.Generate(payloadSeat1)
	if err != nil {
		t.Fatalf("failed to generate QR for seat 1: %v", err)
	}

	// Generate QR for Seat 2
	payloadSeat2 := qrGen.GenerateVerificationPayload(bookingRef, "A-02", eventID)
	qrBytesSeat2, err := qrGen.Generate(payloadSeat2)
	if err != nil {
		t.Fatalf("failed to generate QR for seat 2: %v", err)
	}

	// Verify payloads are unique
	if payloadSeat1 == payloadSeat2 {
		t.Fatalf("expected different payloads for different seats, both were: %s", payloadSeat1)
	}

	// Verify generated PNG byte slices are distinct
	if bytes.Equal(qrBytesSeat1, qrBytesSeat2) {
		t.Fatal("expected different QR code PNG bytes for different seats, but they were identical")
	}

	// Verify deserialization of payloads
	var p1, p2 VerificationPayload
	if err := json.Unmarshal([]byte(payloadSeat1), &p1); err != nil {
		t.Fatalf("failed to unmarshal payload 1: %v", err)
	}
	if err := json.Unmarshal([]byte(payloadSeat2), &p2); err != nil {
		t.Fatalf("failed to unmarshal payload 2: %v", err)
	}

	if p1.SeatID != "A-01" || p2.SeatID != "A-02" {
		t.Fatalf("unmarshalled seats unexpected: seat1=%s, seat2=%s", p1.SeatID, p2.SeatID)
	}
	if p1.BookingRef != bookingRef || p2.BookingRef != bookingRef {
		t.Fatalf("unmarshalled bookingRef mismatch")
	}
	if p1.EventID != eventID || p2.EventID != eventID {
		t.Fatalf("unmarshalled eventID mismatch")
	}
}

func TestPDF_GenerateTicket(t *testing.T) {
	pdfGen := NewTicketPDFGenerator(nil)

	ticketData := TicketPDFData{
		BookingRef:   "TICKID-REF-2026-X8",
		EventTitle:   "Coldplay: Music of the Spheres World Tour Jakarta",
		EventDate:    "Saturday, 15 November 2026, 19:30 WIB",
		VenueName:    "Gelora Bung Karno Main Stadium",
		VenueAddress: "Jl. Pintu Satu Senayan, Gelora, Jakarta Pusat 10270",
		AttendeeName: "Budi Santoso",
		SeatDetails:  "Section VIP-A, Row 03, Seat 14",
		TierName:     "Ultimate VIP Experience",
		TotalAmount:  "3,500,000",
		Currency:     "IDR",
		QRData:       `{"bookingRef":"TICKID-REF-2026-X8","seatId":"VIP-A-14","eventId":"COLDPLAY-JKT"}`,
	}

	pdfBytes, err := pdfGen.GenerateTicket(ticketData)
	if err != nil {
		t.Fatalf("failed to generate ticket PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatal("generated ticket PDF byte slice is empty")
	}

	// Assert %PDF- binary magic header
	if !bytes.HasPrefix(pdfBytes, PDFMagicHeader) {
		t.Fatalf("generated document does not start with %%PDF- magic header, got: %q", string(pdfBytes[:8]))
	}

	// Assert non-empty size (>10KB due to embedded Go fonts and high-resolution QR PNG)
	const minExpectedSize = 10 * 1024 // 10 KB
	if len(pdfBytes) < minExpectedSize {
		t.Fatalf("generated PDF size (%d bytes) is less than expected minimum (%d bytes)", len(pdfBytes), minExpectedSize)
	}

	// Assert EOF marker exists in standard PDF structure
	if !bytes.Contains(pdfBytes, []byte("%%EOF")) {
		t.Fatalf("expected generated PDF to contain %s end-of-file marker", "%%EOF")
	}
}

func TestPDF_MultiTicket(t *testing.T) {
	pdfGen := NewTicketPDFGenerator(nil)

	seats := []struct {
		Seat   string
		Holder string
	}{
		{Seat: "VIP Section A, Row 1, Seat 101", Holder: "Jane Doe"},
		{Seat: "VIP Section A, Row 1, Seat 102", Holder: "John Doe"},
		{Seat: "VIP Section A, Row 1, Seat 103", Holder: "Alice Smith"},
	}

	multiTickets := make([]TicketPDFData, len(seats))
	for i, s := range seats {
		multiTickets[i] = TicketPDFData{
			BookingRef:   "TICKID-GROUP-456",
			EventTitle:   "Java Jazz Festival 2026 - All Access",
			EventDate:    "Sunday, 28 June 2026, 16:00 WIB",
			VenueName:    "JIEXPO Kemayoran",
			VenueAddress: "Gedung Pusat Niaga, Arena PRJ Kemayoran, Jakarta 10620",
			AttendeeName: s.Holder,
			SeatDetails:  s.Seat,
			TierName:     "VIP PASS",
			TotalAmount:  "1,250,000",
			Currency:     "IDR",
			QRData:       fmt.Sprintf(`{"bookingRef":"TICKID-GROUP-456","seat":"%s"}`, s.Seat),
		}
	}

	pdfBytes, err := pdfGen.GenerateMultiTicket(multiTickets)
	if err != nil {
		t.Fatalf("failed to generate multi-ticket PDF: %v", err)
	}

	if !bytes.HasPrefix(pdfBytes, PDFMagicHeader) {
		t.Fatal("multi-ticket PDF missing %PDF- header")
	}

	// A 3-page PDF with embedded fonts and 3 distinct QR codes should be > 20KB
	const minMultiSize = 20 * 1024
	if len(pdfBytes) < minMultiSize {
		t.Fatalf("multi-ticket PDF size (%d bytes) is smaller than expected (%d bytes)", len(pdfBytes), minMultiSize)
	}

	// Verify page markers in PDF structure
	pageCountMarker := []byte("/Type /Page")
	count := bytes.Count(pdfBytes, pageCountMarker)
	if count < 3 {
		t.Fatalf("expected at least 3 page objects in multi-ticket PDF, found %d", count)
	}
}

func TestPDF_InvalidData(t *testing.T) {
	pdfGen := NewTicketPDFGenerator(nil)

	t.Run("empty ticket slice", func(t *testing.T) {
		_, err := pdfGen.GenerateMultiTicket([]TicketPDFData{})
		if err == nil {
			t.Fatal("expected error when generating multi-ticket with empty slice, got nil")
		}
	})

	t.Run("empty fields edge case", func(t *testing.T) {
		// All fields blank or empty strings - should handle gracefully with fallbacks and not panic
		emptyTicket := TicketPDFData{}
		pdfBytes, err := pdfGen.GenerateTicket(emptyTicket)
		if err != nil {
			t.Fatalf("unexpected error generating ticket with empty fields: %v", err)
		}
		if !bytes.HasPrefix(pdfBytes, PDFMagicHeader) {
			t.Fatal("expected valid PDF header even with empty fields")
		}
		if len(pdfBytes) < 5000 {
			t.Fatalf("expected substantial PDF output even with fallbacks, got %d bytes", len(pdfBytes))
		}
	})

	t.Run("extreme text wrapping and special characters", func(t *testing.T) {
		longTicket := TicketPDFData{
			BookingRef:   "REF-SUPER-LONG-ALPHANUMERIC-CODE-THAT-EXCEEDS-NORMAL-WIDTH-BOUNDARIES-999999999",
			EventTitle:   "Annual International Mega Music Festival 2026: The Complete Symphony of Orchestral Rock, Pop, Electronic, and Traditional Fusion Extravaganza Live across 5 Stages",
			EventDate:    "Friday to Sunday, 12-14 December 2026, Doors open at 10:00 AM WIB, Concert until midnight",
			VenueName:    "International Convention and Exhibition Mega Complex Hall 1, 2, 3, 5, 6, 7 and Outdoor Grand Arena",
			VenueAddress: "Boulevard Raya Utama No. 888, Sektor Barat, Kawasan Terpadu Industri Hiburan dan Budaya, Kota Tangerang Selatan, Banten 15339, Republik Indonesia",
			AttendeeName: "Dr. Muhammad Alexander Christopher Van Der Bilt Jr., M.Eng., Ph.D.",
			SeatDetails:  "Tribune Upper Tier Platinum Section East Wing, Row 42, Seat 108 (Near Gate 4B Staircase)",
			TierName:     "PLATINUM TIER ALL-INCLUSIVE VIP LOUNGE ACCESS & BACKSTAGE TOUR",
			TotalAmount:  "125,500,000",
			Currency:     "IDR",
			QRData:       "https://tickid.com/verify?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
		}

		pdfBytes, err := pdfGen.GenerateTicket(longTicket)
		if err != nil {
			t.Fatalf("unexpected error generating ticket with extreme text length: %v", err)
		}
		if !bytes.HasPrefix(pdfBytes, PDFMagicHeader) {
			t.Fatal("expected valid PDF header with long text")
		}
	})

	t.Run("precomputed QRBytes provided", func(t *testing.T) {
		qrGen := NewQRCodeGenerator(256)
		customQR, err := qrGen.Generate("PRECOMPUTED-PAYLOAD-TEST")
		if err != nil {
			t.Fatalf("failed to generate custom QR: %v", err)
		}

		ticketWithQR := TicketPDFData{
			BookingRef:   "TICKID-PRECOMPUTED",
			EventTitle:   "Custom QR Test Event",
			AttendeeName: "Test Attendee",
			QRBytes:      customQR,
		}

		pdfBytes, err := pdfGen.GenerateTicket(ticketWithQR)
		if err != nil {
			t.Fatalf("failed to generate ticket with precomputed QRBytes: %v", err)
		}
		if !bytes.HasPrefix(pdfBytes, PDFMagicHeader) {
			t.Fatal("expected valid PDF header")
		}
	})
}
