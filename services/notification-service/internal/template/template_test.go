package template

import (
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize template engine: %v", err)
	}
	if engine == nil {
		t.Fatal("expected non-nil template engine")
	}
}

func TestRenderOrderConfirmation_Success(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	data := OrderConfirmationData{
		BookingRef:           "TICK-2026-9988",
		CustomerName:         "John Doe",
		CustomerEmail:        "johndoe@example.com",
		EventTitle:           "Coldplay Music of the Spheres World Tour",
		EventDate:            "Saturday, 25 Nov 2026, 20:00 WIB",
		VenueName:            "Gelora Bung Karno Stadium",
		VenueAddress:         "Jl. Pintu Satu Senayan, Jakarta",
		TierName:             "VIP Experience",
		Seats: []SeatItem{
			{SeatID: "s-1", Section: "West VIP", Row: "A", Number: "01", PriceFormatted: "2,500,000"},
			{SeatID: "s-2", Section: "West VIP", Row: "A", Number: "02", PriceFormatted: "2,500,000"},
		},
		TotalAmountFormatted: "5,000,000",
		Currency:             "IDR",
		QRCodeBase64:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
		HasPDFAttachment:     true,
		SupportEmail:         "support@tickid.com",
	}

	result, err := engine.RenderOrderConfirmation(data)
	if err != nil {
		t.Fatalf("failed to render order confirmation: %v", err)
	}

	// 1. Verify HTML Body
	html := result.HTML
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("HTML missing DOCTYPE")
	}
	if !strings.Contains(html, "TICK-2026-9988") {
		t.Errorf("HTML missing booking reference")
	}
	if !strings.Contains(html, "John Doe") {
		t.Errorf("HTML missing customer name")
	}
	if !strings.Contains(html, "Coldplay Music of the Spheres World Tour") {
		t.Errorf("HTML missing event title")
	}
	if !strings.Contains(html, "Gelora Bung Karno Stadium") {
		t.Errorf("HTML missing venue name")
	}
	if !strings.Contains(html, "West VIP") {
		t.Errorf("HTML missing seat section")
	}
	if !strings.Contains(html, "5,000,000 IDR") {
		t.Errorf("HTML missing formatted total amount with currency")
	}
	if !strings.Contains(html, "ticket_TICK-2026-9988.pdf") {
		t.Errorf("HTML missing PDF attachment notice")
	}
	if !strings.Contains(html, "data:image/png;base64,") {
		t.Errorf("HTML missing embedded base64 QR code data URL")
	}
	if !strings.Contains(html, "alt=\"E-Ticket Gate Pass QR Code\"") {
		t.Errorf("HTML missing QR code image element")
	}
	if !strings.Contains(html, "support@tickid.com") {
		t.Errorf("HTML missing support email")
	}
	if !strings.Contains(html, "Secured by TickId Gate Verification Engine") {
		t.Errorf("HTML missing security badge")
	}

	// 2. Verify Plain Text Body
	text := result.Text
	if !strings.Contains(text, "TICKID - ORDER CONFIRMATION & E-TICKET DELIVERY") {
		t.Errorf("Text missing header banner")
	}
	if !strings.Contains(text, "Booking Reference : TICK-2026-9988") {
		t.Errorf("Text missing booking reference")
	}
	if !strings.Contains(text, "John Doe") {
		t.Errorf("Text missing customer name")
	}
	if !strings.Contains(text, "ticket_TICK-2026-9988.pdf") {
		t.Errorf("Text missing PDF notice")
	}
}

func TestRenderPaymentFailed_Success(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	data := PaymentFailedData{
		BookingRef:   "TICK-FAIL-1234",
		CustomerName: "Jane Smith",
		EventTitle:   "Java Jazz Festival 2026",
		Reason:       "Payment transaction timed out after 15 minutes",
		ExpiredAt:    "2026-10-18 21:00:00 WIB",
		RebookURL:    "https://tickid.com/events/java-jazz",
		SupportEmail: "help@tickid.com",
	}

	result, err := engine.RenderPaymentFailed(data)
	if err != nil {
		t.Fatalf("failed to render payment failed template: %v", err)
	}

	// 1. Verify HTML Body
	html := result.HTML
	if !strings.Contains(html, "TICK-FAIL-1234") {
		t.Errorf("HTML missing booking reference")
	}
	if !strings.Contains(html, "Jane Smith") {
		t.Errorf("HTML missing customer name")
	}
	if !strings.Contains(html, "Java Jazz Festival 2026") {
		t.Errorf("HTML missing event title")
	}
	if !strings.Contains(html, "Payment transaction timed out after 15 minutes") {
		t.Errorf("HTML missing reason")
	}
	if !strings.Contains(html, "2026-10-18 21:00:00 WIB") {
		t.Errorf("HTML missing expiration timestamp")
	}
	if !strings.Contains(html, "https://tickid.com/events/java-jazz") {
		t.Errorf("HTML missing re-book CTA URL")
	}
	if !strings.Contains(html, "help@tickid.com") {
		t.Errorf("HTML missing support email")
	}
	if !strings.Contains(html, "Seats Released") {
		t.Errorf("HTML missing released inventory notice")
	}

	// 2. Verify Plain Text Body
	text := result.Text
	if !strings.Contains(text, "TICKID - PAYMENT EXPIRED / INCOMPLETE NOTIFICATION") {
		t.Errorf("Text missing header")
	}
	if !strings.Contains(text, "TICK-FAIL-1234") {
		t.Errorf("Text missing booking reference")
	}
	if !strings.Contains(text, "https://tickid.com/events/java-jazz") {
		t.Errorf("Text missing rebook URL")
	}
}

func TestRenderEventNotice_Success(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	data := EventNoticeData{
		BookingRef:     "TICK-NOTICE-5555",
		CustomerName:   "Alex Johnson",
		EventTitle:     "Summer Rock Fest",
		NoticeType:     "Event Cancelled",
		Reason:         "Extreme weather advisory issued by meteorological agency",
		ScheduleUpdate: "New dates pending evaluation for Spring 2027",
		RefundInfo:     "100% automatic refund will be credited to your original payment method within 5-7 business days",
		SupportEmail:   "support@tickid.com",
	}

	result, err := engine.RenderEventNotice(data)
	if err != nil {
		t.Fatalf("failed to render event notice: %v", err)
	}

	// 1. Verify HTML Body
	html := result.HTML
	if !strings.Contains(html, "Event Cancelled") {
		t.Errorf("HTML missing notice type")
	}
	if !strings.Contains(html, "Summer Rock Fest") {
		t.Errorf("HTML missing event title")
	}
	if !strings.Contains(html, "TICK-NOTICE-5555") {
		t.Errorf("HTML missing booking reference")
	}
	if !strings.Contains(html, "Extreme weather advisory") {
		t.Errorf("HTML missing reason")
	}
	if !strings.Contains(html, "100% automatic refund") {
		t.Errorf("HTML missing refund info")
	}
	if !strings.Contains(html, "Spring 2027") {
		t.Errorf("HTML missing schedule update")
	}

	// 2. Verify Text Body
	text := result.Text
	if !strings.Contains(text, "TICKID - EVENT ADVISORY: Event Cancelled") {
		t.Errorf("Text missing notice header")
	}
	if !strings.Contains(text, "Summer Rock Fest") {
		t.Errorf("Text missing event title")
	}
	if !strings.Contains(text, "100% automatic refund") {
		t.Errorf("Text missing refund details")
	}
}

func TestHTMLEscaping_Security(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	maliciousData := OrderConfirmationData{
		BookingRef:           "<script>alert('xss')</script>",
		CustomerName:         "<b>Malicious User</b>",
		CustomerEmail:        "malicious@example.com",
		EventTitle:           "<img src=x onerror=alert(1)>Event",
		EventDate:            "Tomorrow",
		VenueName:            "<style>body{display:none}</style>Venue",
		TierName:             "General",
		TotalAmountFormatted: "100,000",
		Currency:             "IDR",
	}

	result, err := engine.RenderOrderConfirmation(maliciousData)
	if err != nil {
		t.Fatalf("rendering failed with malicious payload: %v", err)
	}

	html := result.HTML
	// Verify raw scripts and tags are escaped by html/template
	if strings.Contains(html, "<script>alert('xss')</script>") {
		t.Errorf("XSS script tag was not escaped in HTML output")
	}
	if !strings.Contains(html, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;") &&
		!strings.Contains(html, "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;") {
		t.Errorf("Expected escaped script tag in HTML")
	}

	if strings.Contains(html, "<b>Malicious User</b>") {
		t.Errorf("Raw <b> tag was not escaped in HTML output")
	}
	if !strings.Contains(html, "&lt;b&gt;Malicious User&lt;/b&gt;") {
		t.Errorf("Expected escaped <b> tag in HTML output")
	}

	if strings.Contains(html, "<img src=x onerror=alert(1)>") {
		t.Errorf("Raw <img> onerror tag was not escaped in HTML output")
	}
}

func TestDefaultFallbacks(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	// Missing SupportEmail and Currency
	result, err := engine.RenderOrderConfirmation(OrderConfirmationData{
		BookingRef:   "TICK-EMPTY-DEFAULTS",
		CustomerName: "Test User",
		EventTitle:   "Fallback Concert",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(result.HTML, "support@tickid.com") {
		t.Errorf("Expected default support email support@tickid.com")
	}
	if !strings.Contains(result.HTML, "IDR") {
		t.Errorf("Expected default currency IDR")
	}

	// Payment Failed without reason
	pfResult, err := engine.RenderPaymentFailed(PaymentFailedData{
		BookingRef:   "TICK-PF-DEFAULT",
		CustomerName: "Test User",
		EventTitle:   "Expired Concert",
	})
	if err != nil {
		t.Fatalf("payment failed render failed: %v", err)
	}
	if !strings.Contains(pfResult.HTML, "support@tickid.com") {
		t.Errorf("Expected default support email in payment failed")
	}
}

func TestRenderOrderConfirmation_WithoutQR(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	result, err := engine.RenderOrderConfirmation(OrderConfirmationData{
		BookingRef:   "TICK-NO-QR",
		CustomerName: "Alice",
		EventTitle:   "Acoustic Night",
		QRCodeBase64: "", // No QR Code
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if strings.Contains(result.HTML, "<div class=\"qr-section\">") {
		t.Errorf("Expected QR section div to be omitted when QRCodeBase64 is empty")
	}
	if strings.Contains(result.HTML, "alt=\"E-Ticket Gate Pass QR Code\"") {
		t.Errorf("Expected QR image to be omitted when QRCodeBase64 is empty")
	}
}

func TestSafeQR_Helper(t *testing.T) {
	// Empty string
	if safeQR("") != "" {
		t.Errorf("expected empty string for empty input")
	}

	// Raw base64 string without data prefix
	res1 := safeQR("aGVsbG8=")
	if string(res1) != "data:image/png;base64,aGVsbG8=" {
		t.Errorf("expected data URL prefix added, got: %s", res1)
	}

	// Already has data:image prefix
	res2 := safeQR("data:image/png;base64,aGVsbG8=")
	if string(res2) != "data:image/png;base64,aGVsbG8=" {
		t.Errorf("expected preserved data URL, got: %s", res2)
	}
}

