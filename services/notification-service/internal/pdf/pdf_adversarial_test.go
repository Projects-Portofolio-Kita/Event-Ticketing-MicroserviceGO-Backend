package pdf_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/pdf"
	"github.com/makiuchi-d/gozxing"
	zxingqr "github.com/makiuchi-d/gozxing/qrcode"
	rscpdf "rsc.io/pdf"
)

// Helper to decode QR code PNG bytes into payload string using gozxing
func decodeQRCodePNG(t *testing.T, pngBytes []byte) string {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("failed to decode PNG image from QR bytes: %v", err)
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatalf("failed to create binary bitmap from QR image: %v", err)
	}

	reader := zxingqr.NewQRCodeReader()
	result, err := reader.Decode(bmp, nil)
	if err != nil {
		t.Fatalf("failed to decode QR barcode data: %v", err)
	}

	return result.GetText()
}

// 1. Adversarial QR Code Payload Decoding & Format Round-Trip
func TestAdversarial_QRPayloadDecodeAndVerification(t *testing.T) {
	qrGen := pdf.NewQRCodeGenerator(256)

	testCases := []struct {
		name       string
		bookingRef string
		seatID     string
		eventID    string
	}{
		{
			name:       "Standard Alphanumeric",
			bookingRef: "TICKID-REF-2026-X8",
			seatID:     "SEAT-VIP-A1",
			eventID:    "EVT-COLDPLAY-JKT",
		},
		{
			name:       "UUID Identifiers",
			bookingRef: "c6a7936d-1bf8-466d-b2a8-a53d1df4c6f9",
			seatID:     "7e0129a3-5c21-4770-98d1-d249eb403061",
			eventID:    "e37171e0-7988-469b-9804-b590e8c2534a",
		},
		{
			name:       "Special Characters in IDs",
			bookingRef: "REF#99-88/2026:WIB",
			seatID:     "ROW_A/SEAT#01+02",
			eventID:    "EVT:FESTIVAL_2026&SPECIAL",
		},
		{
			name:       "Unicode and Diacritics in Identifiers",
			bookingRef: "BÜCHUNG-RÉF-77",
			seatID:     "SIÈGE-N°42",
			eventID:    "ÉVÉNEMENT-NOËL",
		},
		{
			name:       "Long Reference Identifiers",
			bookingRef: strings.Repeat("B", 64),
			seatID:     strings.Repeat("S", 64),
			eventID:    strings.Repeat("E", 64),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := qrGen.GenerateVerificationPayload(tc.bookingRef, tc.seatID, tc.eventID)
			if payload == "" {
				t.Fatalf("generated verification payload is empty")
			}

			// Generate QR PNG bytes
			qrPNG, err := qrGen.Generate(payload)
			if err != nil {
				t.Fatalf("failed to generate QR PNG: %v", err)
			}

			// Decode using independent barcode reader
			decodedText := decodeQRCodePNG(t, qrPNG)
			if decodedText != payload {
				t.Fatalf("mismatch in decoded QR payload.\nExpected: %s\nGot:      %s", payload, decodedText)
			}

			// Parse and verify JSON structure {bookingRef, seatId, eventId}
			var parsed pdf.VerificationPayload
			if err := json.Unmarshal([]byte(decodedText), &parsed); err != nil {
				t.Fatalf("failed to parse decoded QR payload as JSON: %v", err)
			}

			if parsed.BookingRef != tc.bookingRef {
				t.Errorf("bookingRef mismatch: expected %q, got %q", tc.bookingRef, parsed.BookingRef)
			}
			if parsed.SeatID != tc.seatID {
				t.Errorf("seatId mismatch: expected %q, got %q", tc.seatID, parsed.SeatID)
			}
			if parsed.EventID != tc.eventID {
				t.Errorf("eventId mismatch: expected %q, got %q", tc.eventID, parsed.EventID)
			}
		})
	}
}

// 2. Multi-Seat Orders with Large Seat Counts (10 to 20 seats) generating multi-page PDFs
func TestAdversarial_MultiSeatPDFScaling(t *testing.T) {
	qrGen := pdf.NewQRCodeGenerator(256)
	pdfGen := pdf.NewTicketPDFGenerator(qrGen)

	seatCounts := []int{5, 10, 15, 20}

	for _, count := range seatCounts {
		t.Run(fmt.Sprintf("%d_Seats", count), func(t *testing.T) {
			start := time.Now()
			tickets := make([]pdf.TicketPDFData, count)
			expectedPayloads := make([]string, count)

			for i := 0; i < count; i++ {
				seatID := fmt.Sprintf("SEAT-%03d", i+1)
				bookingRef := fmt.Sprintf("BOOK-MULTI-%04d", count)
				eventID := "EVT-STADIUM-ROCK"
				payload := qrGen.GenerateVerificationPayload(bookingRef, seatID, eventID)
				expectedPayloads[i] = payload

				tickets[i] = pdf.TicketPDFData{
					BookingRef:   bookingRef,
					EventTitle:   "Massive Stadium Rock Tour 2026",
					EventDate:    "Saturday, 28 November 2026, 19:00 WIB",
					VenueName:    "Gelora Bung Karno Stadium",
					VenueAddress: "Jl. Pintu Satu Senayan, Gelora, Jakarta",
					AttendeeName: fmt.Sprintf("VIP Guest #%d", i+1),
					SeatDetails:  fmt.Sprintf("Section VIP-Tribune, Row 05, Seat %d", i+1),
					TierName:     "VIP Diamond",
					TotalAmount:  "3,500,000",
					Currency:     "IDR",
					QRData:       payload,
				}
			}

			pdfBytes, err := pdfGen.GenerateMultiTicket(tickets)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("failed to generate %d-seat multi-ticket PDF: %v", count, err)
			}

			// Validate PDF header magic
			if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
				t.Fatalf("output does not have valid PDF magic header")
			}

			// Validate page count using rsc.io/pdf reader
			pdfReader, err := rscpdf.NewReader(bytes.NewReader(pdfBytes), int64(len(pdfBytes)))
			if err != nil {
				t.Fatalf("rsc.io/pdf failed to parse generated PDF: %v", err)
			}

			numPages := pdfReader.NumPage()
			if numPages != count {
				t.Fatalf("page count mismatch: expected %d pages, got %d pages", count, numPages)
			}

			// Performance check: generating 20 tickets with 20 distinct QR codes should take < 3 seconds
			t.Logf("Generated %d-page PDF in %v (size: %d bytes, %.1f KB/page)", count, elapsed, len(pdfBytes), float64(len(pdfBytes))/float64(count)/1024.0)
			if elapsed > 5*time.Second {
				t.Errorf("PDF generation took too long: %v for %d seats", elapsed, count)
			}
		})
	}
}

// 3. Stress-Test Unicode, Emojis, and Special Characters in PDF Generation
func TestAdversarial_PDFUnicodeAndSpecialCharacters(t *testing.T) {
	qrGen := pdf.NewQRCodeGenerator(256)
	pdfGen := pdf.NewTicketPDFGenerator(qrGen)

	testCases := []struct {
		name         string
		eventTitle   string
		venueName    string
		venueAddress string
		attendeeName string
		tierName     string
		seatDetails  string
	}{
		{
			name:         "Latin Accented & Diacritics",
			eventTitle:   "Concert de Musique Classique: Noël & Café de Paris",
			venueName:    "Théâtre des Champs-Élysées",
			venueAddress: "15 Avenue Montaigne, 75008 Paris, France",
			attendeeName: "Élodie François Müller-Jäger",
			tierName:     "Carré d'Or",
			seatDetails:  "Rangée A, Siège 01",
		},
		{
			name:         "German & Nordic Characters",
			eventTitle:   "Großes Frühlingskonzert & Håkan Hellström Live",
			venueName:    "Kölner Philharmonie & Malmö Arena",
			venueAddress: "Bischofsgartenstraße 1, 50667 Köln, Deutschland",
			attendeeName: "Søren Kierkegaard & Jürgen Klopp",
			tierName:     "Frühbucher Rabatt",
			seatDetails:  "Platz 99 (Gäste)",
		},
		{
			name:         "Emojis in Title and Name",
			eventTitle:   "Summer Music Festival 2026 🌟🎸🔥 (VIP Access 🎟️)",
			venueName:    "Carnival Park 🎪🎡",
			venueAddress: "Jl. Pantai Indah Kapuk No. 8, Jakarta 🇮🇩",
			attendeeName: "Party Lover 🎉🚀",
			tierName:     "VIP Diamond 💎",
			seatDetails:  "Row VIP 👑 Seat 01",
		},
		{
			name:         "CJK & Non-Latin Scripts",
			eventTitle:   "Tokyo Symphony Orchestra 2026 (東京交響楽団 / 서울 공연)",
			venueName:    "Suntory Hall (サントリーホール)",
			venueAddress: "1-13-1 Akasaka, Minato-ku, Tokyo 107-8403",
			attendeeName: "山田 太郎 (Yamada Taro)",
			tierName:     "S席 VIP",
			seatDetails:  "1階 12列 24番",
		},
		{
			name:         "Arabic & Hebrew Non-Latin Scripts",
			eventTitle:   "مهرجان الموسيقى الدولي 2026 / פסטיבל המוזיקה",
			venueName:    "دار الأوبرا المصرية",
			venueAddress: "Gezira Island, Cairo, Egypt",
			attendeeName: "أحمد بن علي (Ahmed bin Ali)",
			tierName:     "درجة أولى",
			seatDetails:  "مقعد 15",
		},
		{
			name:         "Raw HTML and XSS Script Injection Ingestion",
			eventTitle:   "<script>alert('xss')</script> Concert <b>2026</b>",
			venueName:    "<img src=x onerror=alert(1)> Main Stadium",
			venueAddress: "'; DROP TABLE venues; --",
			attendeeName: "\"><script>alert('pwned')</script>",
			tierName:     "<style>body{display:none}</style>",
			seatDetails:  "<iframe src='javascript:alert(1)'></iframe>",
		},
		{
			name:         "Extreme Text Length and Wrapping",
			eventTitle:   strings.Repeat("Ultra Long Event Title Designed to Stress Gopdf Text Wrapping Engine ", 8),
			venueName:    strings.Repeat("Super Detailed Complex Stadium Arena Building Name ", 6),
			venueAddress: strings.Repeat("Long Avenue Address With Sector Details Postal Codes And Notes ", 5),
			attendeeName: strings.Repeat("Prof. Dr. Multi-Hyphenated Ultra Extended Full Legal Name ", 4),
			tierName:     "Extremely Long Tier Category With Package Inclusions",
			seatDetails:  strings.Repeat("Seat-Details-Repeated-Buffer-Block-", 5),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ticketData := pdf.TicketPDFData{
				BookingRef:   "TICK-UNICODE-99",
				EventTitle:   tc.eventTitle,
				VenueName:    tc.venueName,
				VenueAddress: tc.venueAddress,
				AttendeeName: tc.attendeeName,
				TierName:     tc.tierName,
				SeatDetails:  tc.seatDetails,
				TotalAmount:  "1,500,000",
				Currency:     "IDR",
				QRData:       `{"bookingRef":"TICK-UNICODE-99","seatId":"S-1","eventId":"E-1"}`,
			}

			// Must not panic and must produce a valid PDF
			pdfBytes, err := pdfGen.GenerateTicket(ticketData)
			if err != nil {
				t.Fatalf("GenerateTicket unexpectedly errored: %v", err)
			}

			if len(pdfBytes) == 0 {
				t.Fatalf("generated PDF byte slice is empty")
			}

			if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
				t.Fatalf("output does not have valid PDF magic header")
			}

			// Verify PDF is readable by a PDF parser without corruption
			pdfReader, err := rscpdf.NewReader(bytes.NewReader(pdfBytes), int64(len(pdfBytes)))
			if err != nil {
				t.Fatalf("rsc.io/pdf failed to parse generated PDF with unicode inputs: %v", err)
			}

			if pdfReader.NumPage() != 1 {
				t.Fatalf("expected 1 page, got %d", pdfReader.NumPage())
			}
		})
	}
}

// 4. Extremely Large and Boundary Ticket Prices in PDF
func TestAdversarial_PDFPriceBoundaries(t *testing.T) {
	qrGen := pdf.NewQRCodeGenerator(256)
	pdfGen := pdf.NewTicketPDFGenerator(qrGen)

	priceCases := []struct {
		name        string
		totalAmount string
		currency    string
	}{
		{
			name:        "0 IDR Free Ticket",
			totalAmount: "0",
			currency:    "IDR",
		},
		{
			name:        "1 IDR Nominal Ticket",
			totalAmount: "1",
			currency:    "IDR",
		},
		{
			name:        "1 Billion IDR Ultra VIP",
			totalAmount: "1,000,000,000",
			currency:    "IDR",
		},
		{
			name:        "999 Billion IDR Extreme Boundary",
			totalAmount: "999,999,999,999",
			currency:    "IDR",
		},
		{
			name:        "USD with Cents",
			totalAmount: "1,250.75",
			currency:    "USD",
		},
		{
			name:        "Negative Price Edge Case",
			totalAmount: "-50,000",
			currency:    "IDR",
		},
	}

	for _, pc := range priceCases {
		t.Run(pc.name, func(t *testing.T) {
			ticketData := pdf.TicketPDFData{
				BookingRef:   "TICK-PRICE-TEST",
				EventTitle:   "Economic Summit 2026",
				AttendeeName: "Investor Delegate",
				TierName:     "VIP Delegate",
				SeatDetails:  "Plenary Row 1 Seat 01",
				TotalAmount:  pc.totalAmount,
				Currency:     pc.currency,
				QRData:       `{"bookingRef":"TICK-PRICE-TEST","seatId":"S-1","eventId":"E-1"}`,
			}

			pdfBytes, err := pdfGen.GenerateTicket(ticketData)
			if err != nil {
				t.Fatalf("GenerateTicket failed for price %s %s: %v", pc.currency, pc.totalAmount, err)
			}

			if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
				t.Fatalf("invalid PDF header for price %s %s", pc.currency, pc.totalAmount)
			}
		})
	}
}
