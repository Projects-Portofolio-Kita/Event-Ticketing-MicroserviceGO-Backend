package template_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/template"
)

// Polyglot and aggressive XSS vectors to challenge template escaping
var adversarialXSSVectors = []struct {
	name    string
	payload string
}{
	{
		name:    "Standard Script Tag",
		payload: "<script>alert('XSS-1')</script>",
	},
	{
		name:    "Attribute Breakout with Event Handler",
		payload: `"><img src=x onerror="alert('XSS-2')">`,
	},
	{
		name:    "SVG Onload Vector",
		payload: `<svg/onload=alert('XSS-3')>`,
	},
	{
		name:    "IFrame Injection",
		payload: `<iframe src="javascript:alert('XSS-4')"></iframe>`,
	},
	{
		name:    "Style Block Injection to Deface Email",
		payload: `<style>body{display:none !important;}</style>`,
	},
	{
		name:    "Javascript Pseudo-Protocol",
		payload: `javascript:alert('XSS-5')`,
	},
	{
		name:    "HTML Entity Obfuscation Attempt",
		payload: `&lt;script&gt;alert('XSS-6')&lt;/script&gt;`,
	},
	{
		name:    "Context-breaking Polyglot Vector",
		payload: `'"><script src=//evil.com/xss.js></script>`,
	},
}

// 1. Stress-test safeQR implementation against malicious QR data URIs
func TestAdversarial_SafeQR_Security(t *testing.T) {
	engine, err := template.NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	testCases := []struct {
		name          string
		qrInput       string
		mustNotContain []string
	}{
		{
			name:    "Javascript Scheme Injected in QR",
			qrInput: "javascript:alert('pwned')",
			mustNotContain: []string{
				"src=\"javascript:",
				"src='javascript:",
			},
		},
		{
			name:    "HTML Script Tag Injected in QR",
			qrInput: "<script>alert('qr-xss')</script>",
			mustNotContain: []string{
				"<script>",
				"alert('qr-xss')",
			},
		},
		{
			name:    "SVG Image with Embedded Onload",
			qrInput: "data:image/svg+xml;utf8,<svg onload=alert(1)>",
			mustNotContain: []string{
				"<svg onload=alert(1)>", // Must be escaped by html/template
			},
		},
		{
			name:    "Attribute Escape Attempt in Base64",
			qrInput: `data:image/png;base64," onmouseover="alert('xss')`,
			mustNotContain: []string{
				`onmouseover="alert`,
				`onmouseover='alert`,
			},
		},
		{
			name:    "Data HTML Scheme Attempt",
			qrInput: "data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==",
			mustNotContain: []string{
				"src=\"data:text/html",
			},
		},
		{
			name:    "Valid Base64 PNG without prefix",
			qrInput: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
			mustNotContain: []string{
				"<script>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := engine.RenderOrderConfirmation(template.OrderConfirmationData{
				BookingRef:   "TICK-SEC-01",
				CustomerName: "Security Tester",
				EventTitle:   "Security Conference 2026",
				QRCodeBase64: tc.qrInput,
			})
			if err != nil {
				t.Fatalf("unexpected render error: %v", err)
			}

			for _, badPattern := range tc.mustNotContain {
				if strings.Contains(res.HTML, badPattern) {
					t.Errorf("Security violation: rendered HTML contains unescaped malicious pattern %q\nHTML snippet around QR:\n%s",
						badPattern, extractQRContext(res.HTML))
				}
			}
		})
	}
}

// 2. Comprehensive HTML Escaping on All User Fields Across All 3 Templates
func TestAdversarial_TemplateFieldEscaping(t *testing.T) {
	engine, err := template.NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	for _, vec := range adversarialXSSVectors {
		t.Run("OrderConfirmation_"+vec.name, func(t *testing.T) {
			data := template.OrderConfirmationData{
				BookingRef:           vec.payload,
				CustomerName:         vec.payload,
				CustomerEmail:        "victim@example.com",
				EventTitle:           vec.payload,
				EventDate:            vec.payload,
				VenueName:            vec.payload,
				VenueAddress:         vec.payload,
				TierName:             vec.payload,
				TotalAmountFormatted: vec.payload,
				Currency:             vec.payload,
				Seats: []template.SeatItem{
					{
						SeatID:         vec.payload,
						Section:        vec.payload,
						Row:            vec.payload,
						Number:         vec.payload,
						PriceFormatted: vec.payload,
					},
				},
				DownloadURL:  vec.payload,
				SupportEmail: vec.payload,
			}

			res, err := engine.RenderOrderConfirmation(data)
			if err != nil {
				t.Fatalf("render failed with vector %s: %v", vec.name, err)
			}

			// Disallow raw unescaped script / iframe / svg tags
			assertNoRawExecutableHTML(t, res.HTML, vec.name)
		})

		t.Run("PaymentFailed_"+vec.name, func(t *testing.T) {
			data := template.PaymentFailedData{
				BookingRef:   vec.payload,
				CustomerName: vec.payload,
				EventTitle:   vec.payload,
				Reason:       vec.payload,
				ExpiredAt:    vec.payload,
				RebookURL:    vec.payload,
				SupportEmail: vec.payload,
			}

			res, err := engine.RenderPaymentFailed(data)
			if err != nil {
				t.Fatalf("render failed with vector %s: %v", vec.name, err)
			}

			assertNoRawExecutableHTML(t, res.HTML, vec.name)
		})

		t.Run("EventNotice_"+vec.name, func(t *testing.T) {
			data := template.EventNoticeData{
				BookingRef:     vec.payload,
				CustomerName:   vec.payload,
				EventTitle:     vec.payload,
				NoticeType:     vec.payload,
				Reason:         vec.payload,
				ScheduleUpdate: vec.payload,
				RefundInfo:     vec.payload,
				SupportEmail:   vec.payload,
			}

			res, err := engine.RenderEventNotice(data)
			if err != nil {
				t.Fatalf("render failed with vector %s: %v", vec.name, err)
			}

			assertNoRawExecutableHTML(t, res.HTML, vec.name)
		})
	}
}

// 3. Multi-Seat Email Rendering (10-20 Seats)
func TestAdversarial_TemplateMultiSeatTable(t *testing.T) {
	engine, err := template.NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	counts := []int{10, 15, 20}
	for _, count := range counts {
		t.Run(fmt.Sprintf("%d_Seats", count), func(t *testing.T) {
			seats := make([]template.SeatItem, count)
			for i := 0; i < count; i++ {
				seats[i] = template.SeatItem{
					SeatID:         fmt.Sprintf("seat-%03d", i+1),
					Section:        fmt.Sprintf("Tribune %c", 'A'+(i%4)),
					Row:            fmt.Sprintf("Row-%02d", (i/4)+1),
					Number:         fmt.Sprintf("N-%03d", i+1),
					PriceFormatted: "500,000",
				}
			}

			data := template.OrderConfirmationData{
				BookingRef:           fmt.Sprintf("BOOK-MULTI-%d", count),
				CustomerName:         "Corporate Group Booker",
				CustomerEmail:        "corp@tickid.com",
				EventTitle:           "National Tech Summit 2026",
				EventDate:            "Friday, 10 Dec 2026, 09:00 WIB",
				VenueName:            "Jakarta Convention Center",
				TierName:             "Corporate Pass",
				Seats:                seats,
				TotalAmountFormatted: fmt.Sprintf("%d,000,000", count/2),
				Currency:             "IDR",
				QRCodeBase64:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
			}

			res, err := engine.RenderOrderConfirmation(data)
			if err != nil {
				t.Fatalf("render failed for %d seats: %v", count, err)
			}

			// Verify seat count label in HTML
			expectedHeader := fmt.Sprintf("Seats Reserved (%d)", count)
			if !strings.Contains(res.HTML, expectedHeader) {
				t.Errorf("HTML missing expected seat count header: %s", expectedHeader)
			}

			// Verify every seat number appears in HTML and text
			for i := 0; i < count; i++ {
				seatNum := fmt.Sprintf("N-%03d", i+1)
				if !strings.Contains(res.HTML, seatNum) {
					t.Errorf("HTML missing seat number %s in table", seatNum)
				}
				if !strings.Contains(res.Text, seatNum) {
					t.Errorf("Plain text missing seat number %s", seatNum)
				}
			}
		})
	}
}

// 4. Unicode, Emojis, and Boundary Character Sets in Templates
func TestAdversarial_TemplateUnicodeAndEmojis(t *testing.T) {
	engine, err := template.NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	testCases := []struct {
		name         string
		customerName string
		eventTitle   string
		venueName    string
		venueAddress string
		reason       string
	}{
		{
			name:         "Emojis in All Fields",
			customerName: "Alex 🎸🔥",
			eventTitle:   "Super Concert 2026 🌟🎟️🚀",
			venueName:    "Grand Arena 🏟️",
			venueAddress: "Jakarta 🇮🇩",
			reason:       "Payment timeout ⏱️❌",
		},
		{
			name:         "French & German Diacritics",
			customerName: "François & Élodie Müller",
			eventTitle:   "Fête de la Musique & Frühlingsfest",
			venueName:    "Théâtre de l'Opéra",
			venueAddress: "Boulevard des Capucines, 75009 Paris",
			reason:       "Carte bancaire expirée",
		},
		{
			name:         "Japanese & Korean CJK",
			customerName: "佐藤 健 (Sato Takeru)",
			eventTitle:   "東京ジャズフェスティバル 2026 (Tokyo Jazz Festival / 서울 콘서트)",
			venueName:    "東京国際フォーラム",
			venueAddress: "東京都千代田区丸の内3丁目5番1号",
			reason:       "決済タイムアウト",
		},
		{
			name:         "Arabic Script",
			customerName: "محمد عبد الله",
			eventTitle:   "مهرجان الموسيقى العربية 2026",
			venueName:    "دار الأوبرا المصرية",
			venueAddress: "القاهرة، مصر",
			reason:       "انتهت مهلة الدفع",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Order Confirmation
			ocRes, err := engine.RenderOrderConfirmation(template.OrderConfirmationData{
				BookingRef:   "TICK-UNI-123",
				CustomerName: tc.customerName,
				EventTitle:   tc.eventTitle,
				VenueName:    tc.venueName,
				VenueAddress: tc.venueAddress,
			})
			if err != nil {
				t.Fatalf("order confirmation failed on unicode: %v", err)
			}
			// In HTML, & is escaped to &amp;, while unicode characters are preserved
			escapedName := strings.ReplaceAll(tc.customerName, "&", "&amp;")
			if !strings.Contains(ocRes.HTML, escapedName) {
				t.Errorf("HTML missing unicode customer name: %s", escapedName)
			}
			if !strings.Contains(ocRes.Text, tc.customerName) {
				t.Errorf("Text missing unicode customer name: %s", tc.customerName)
			}

			// Payment Failed
			pfRes, err := engine.RenderPaymentFailed(template.PaymentFailedData{
				BookingRef:   "TICK-UNI-123",
				CustomerName: tc.customerName,
				EventTitle:   tc.eventTitle,
				Reason:       tc.reason,
			})
			if err != nil {
				t.Fatalf("payment failed render failed on unicode: %v", err)
			}
			escapedReason := strings.ReplaceAll(tc.reason, "&", "&amp;")
			if !strings.Contains(pfRes.HTML, escapedReason) {
				t.Errorf("HTML missing unicode reason: %s", escapedReason)
			}

			// Event Notice
			enRes, err := engine.RenderEventNotice(template.EventNoticeData{
				BookingRef:   "TICK-UNI-123",
				CustomerName: tc.customerName,
				EventTitle:   tc.eventTitle,
				Reason:       tc.reason,
			})
			if err != nil {
				t.Fatalf("event notice render failed on unicode: %v", err)
			}
			escapedTitle := strings.ReplaceAll(tc.eventTitle, "&", "&amp;")
			if !strings.Contains(enRes.HTML, escapedTitle) {
				t.Errorf("HTML missing unicode event title: %s", escapedTitle)
			}
		})
	}
}

// 5. Large & Boundary Ticket Prices in Template
func TestAdversarial_TemplatePriceBoundaries(t *testing.T) {
	engine, err := template.NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	priceCases := []struct {
		name      string
		formatted string
		currency  string
	}{
		{
			name:      "0 IDR Free RSVP",
			formatted: "0",
			currency:  "IDR",
		},
		{
			name:      "1 Billion IDR",
			formatted: "1,000,000,000",
			currency:  "IDR",
		},
		{
			name:      "999 Billion IDR",
			formatted: "999,999,999,999",
			currency:  "IDR",
		},
		{
			name:      "USD with Decimals",
			formatted: "12,500.50",
			currency:  "USD",
		},
	}

	for _, pc := range priceCases {
		t.Run(pc.name, func(t *testing.T) {
			res, err := engine.RenderOrderConfirmation(template.OrderConfirmationData{
				BookingRef:           "TICK-PRICE-1",
				CustomerName:         "VIP Buyer",
				EventTitle:           "Global Gala",
				TotalAmountFormatted: pc.formatted,
				Currency:             pc.currency,
			})
			if err != nil {
				t.Fatalf("render failed for price: %v", err)
			}

			expectedPriceText := fmt.Sprintf("%s %s", pc.formatted, pc.currency)
			if !strings.Contains(res.HTML, expectedPriceText) {
				t.Errorf("HTML missing formatted price %q", expectedPriceText)
			}
			if !strings.Contains(res.Text, expectedPriceText) {
				t.Errorf("Plain text missing formatted price %q", expectedPriceText)
			}
		})
	}
}

// Helper functions for security assertions
func assertNoRawExecutableHTML(t *testing.T, html, vectorName string) {
	t.Helper()
	dangerousSubstrings := []string{
		"<script>",
		"<script ",
		"alert('XSS",
		"alert(\"XSS",
		"<svg/onload",
		"<iframe",
		"<style>body{display:none",
	}

	for _, d := range dangerousSubstrings {
		if strings.Contains(html, d) {
			t.Errorf("Vector %q resulted in unescaped dangerous HTML tag or attribute in output: %s", vectorName, d)
		}
	}
}

func extractQRContext(html string) string {
	idx := strings.Index(html, "qr-section")
	if idx == -1 {
		return "qr-section not found in HTML"
	}
	start := idx - 20
	if start < 0 {
		start = 0
	}
	end := idx + 300
	if end > len(html) {
		end = len(html)
	}
	return html[start:end]
}
