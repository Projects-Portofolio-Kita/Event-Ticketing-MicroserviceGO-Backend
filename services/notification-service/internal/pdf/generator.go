package pdf

import (
	"fmt"
	"strings"

	"github.com/signintech/gopdf"
)

// TicketPDFData holds the comprehensive event, attendee, and seating information
// required to render a branded TickId e-ticket.
type TicketPDFData struct {
	BookingRef   string `json:"booking_ref"`
	EventTitle   string `json:"event_title"`
	EventDate    string `json:"event_date"`
	VenueName    string `json:"venue_name"`
	VenueAddress string `json:"venue_address"`
	AttendeeName string `json:"attendee_name"`
	SeatDetails  string `json:"seat_details"`
	TierName     string `json:"tier_name"`
	TotalAmount  string `json:"total_amount"`
	Currency     string `json:"currency,omitempty"`
	QRData       string `json:"qr_data,omitempty"`
	QRBytes      []byte `json:"-"`
}

// PDFGenerator defines the interface for generating e-ticket PDF documents.
type PDFGenerator interface {
	GenerateTicket(data TicketPDFData) ([]byte, error)
	GenerateMultiTicket(data []TicketPDFData) ([]byte, error)
}

// TicketPDFGenerator implements PDFGenerator using signintech/gopdf.
type TicketPDFGenerator struct {
	qrGen QRCodeGenerator
}

// NewTicketPDFGenerator creates a new TicketPDFGenerator.
// If qrGen is nil, a default QRGenerator is instantiated.
func NewTicketPDFGenerator(qrGen QRCodeGenerator) *TicketPDFGenerator {
	if qrGen == nil {
		qrGen = NewQRCodeGenerator(DefaultQRSize)
	}
	return &TicketPDFGenerator{
		qrGen: qrGen,
	}
}

// GenerateTicket produces a single-ticket PDF document in-memory.
func (g *TicketPDFGenerator) GenerateTicket(data TicketPDFData) ([]byte, error) {
	return g.GenerateMultiTicket([]TicketPDFData{data})
}

// GenerateMultiTicket produces a multi-ticket PDF document containing one page per ticket.
func (g *TicketPDFGenerator) GenerateMultiTicket(data []TicketPDFData) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("ticket data slice cannot be empty")
	}

	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	if err := RegisterEmbeddedFonts(pdf); err != nil {
		return nil, fmt.Errorf("failed to register embedded fonts: %w", err)
	}

	for i, item := range data {
		pdf.AddPage()
		if err := g.renderTicketPage(pdf, item, i+1, len(data)); err != nil {
			return nil, fmt.Errorf("failed to render ticket page %d: %w", i+1, err)
		}
	}

	pdfBytes := pdf.GetBytesPdf()
	if len(pdfBytes) == 0 {
		return nil, fmt.Errorf("generated pdf is empty")
	}

	return pdfBytes, nil
}

// renderTicketPage renders a single ticket onto the current PDF page.
func (g *TicketPDFGenerator) renderTicketPage(pdf *gopdf.GoPdf, data TicketPDFData, pageNum, totalPages int) error {
	// 1. Outer Container Card
	pdf.SetStrokeColor(203, 213, 225) // slate-300
	pdf.SetLineWidth(1)
	pdf.SetFillColor(255, 255, 255) // white
	pdf.RectFromUpperLeftWithStyle(30, 25, 535, 785, "FD")

	// 2. Header Banner
	pdf.SetFillColor(30, 41, 59) // slate-800
	pdf.RectFromUpperLeftWithStyle(30, 25, 535, 70, "F")

	// Brand Accent Top Stripe
	pdf.SetFillColor(99, 102, 241) // indigo-500
	pdf.RectFromUpperLeftWithStyle(30, 25, 535, 4, "F")

	// Brand Title
	pdf.SetTextColor(255, 255, 255)
	if err := pdf.SetFont(FontFamilyBold, "", 20); err != nil {
		return err
	}
	pdf.SetXY(45, 38)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 200, H: 24}, "TickId", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Header Subtitle
	pdf.SetTextColor(199, 210, 254) // indigo-200
	if err := pdf.SetFont(FontFamilyRegular, "", 10); err != nil {
		return err
	}
	pdf.SetXY(45, 64)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 200, H: 16}, "OFFICIAL EVENT E-TICKET", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Booking Reference Badge
	pdf.SetFillColor(49, 46, 129) // indigo-900
	pdf.RectFromUpperLeftWithStyle(355, 40, 195, 36, "F")

	pdf.SetTextColor(224, 231, 255) // indigo-100
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil {
		return err
	}
	pdf.SetXY(362, 44)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 180, H: 12}, "BOOKING REFERENCE", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	bookingRefText := data.BookingRef
	if bookingRefText == "" {
		bookingRefText = "N/A"
	}
	pdf.SetTextColor(255, 255, 255)
	if err := pdf.SetFont(FontFamilyBold, "", 11); err != nil {
		return err
	}
	pdf.SetXY(362, 58)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 180, H: 14}, bookingRefText, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// 3. Event Information Card
	eventY := 105.0
	pdf.SetFillColor(248, 250, 252)   // slate-50
	pdf.SetStrokeColor(226, 232, 240) // slate-200
	pdf.RectFromUpperLeftWithStyle(45, eventY, 505, 125, "FD")

	// Section Label
	pdf.SetTextColor(99, 102, 241) // indigo-500
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(55, eventY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 485, H: 14}, "EVENT DETAILS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Event Title (larger font with fallback wrapping)
	eventTitle := data.EventTitle
	if eventTitle == "" {
		eventTitle = "Untitled Event"
	}
	pdf.SetTextColor(15, 23, 42) // slate-900
	if err := pdf.SetFont(FontFamilyBold, "", 16); err != nil {
		return err
	}
	pdf.SetXY(55, eventY+24)
	_ = pdf.MultiCell(&gopdf.Rect{W: 485, H: 20}, eventTitle)

	// Date & Time
	pdf.SetTextColor(100, 116, 139) // slate-500
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(55, eventY+68)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 80, H: 14}, "DATE & TIME:", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	eventDate := data.EventDate
	if eventDate == "" {
		eventDate = "TBA"
	}
	pdf.SetTextColor(30, 41, 59) // slate-800
	if err := pdf.SetFont(FontFamilyRegular, "", 10); err != nil {
		return err
	}
	pdf.SetXY(135, eventY+68)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 405, H: 14}, eventDate, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Venue
	pdf.SetTextColor(100, 116, 139) // slate-500
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(55, eventY+86)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 80, H: 14}, "VENUE:", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	venueFull := data.VenueName
	if data.VenueAddress != "" {
		if venueFull != "" {
			venueFull += " — " + data.VenueAddress
		} else {
			venueFull = data.VenueAddress
		}
	}
	if venueFull == "" {
		venueFull = "Venue location to be announced"
	}
	pdf.SetTextColor(30, 41, 59) // slate-800
	if err := pdf.SetFont(FontFamilyRegular, "", 10); err != nil {
		return err
	}
	pdf.SetXY(135, eventY+86)
	_ = pdf.MultiCell(&gopdf.Rect{W: 405, H: 15}, venueFull)

	// 4. Attendee, Seating & QR Split Section
	splitY := 240.0
	cardHeight := 280.0

	// Left Box: Attendee & Seating Info (Width: 295)
	pdf.SetFillColor(248, 250, 252)
	pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(45, splitY, 295, cardHeight, "FD")

	// Left Section Label
	pdf.SetTextColor(99, 102, 241) // indigo-500
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+10)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 14}, "ATTENDEE & TICKET SPECIFICATIONS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Attendee Name
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+30)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 12}, "ATTENDEE NAME", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	attendeeName := data.AttendeeName
	if attendeeName == "" {
		attendeeName = "Valued Guest"
	}
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 13); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+44)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 18}, attendeeName, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Ticket Tier
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+68)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 12}, "TICKET TIER", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	tierName := strings.ToUpper(data.TierName)
	if tierName == "" {
		tierName = "GENERAL ADMISSION"
	}
	pdf.SetFillColor(224, 231, 255) // indigo-100
	pdf.RectFromUpperLeftWithStyle(55, splitY+82, 160, 22, "F")
	pdf.SetTextColor(67, 56, 202) // indigo-700
	if err := pdf.SetFont(FontFamilyBold, "", 10); err != nil {
		return err
	}
	pdf.SetXY(60, splitY+82)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 150, H: 22}, tierName, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	// Seat Details
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+112)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 12}, "SEAT / ENTRY ACCESS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	seatDetails := data.SeatDetails
	if seatDetails == "" {
		seatDetails = "Free Seating / General Entry"
	}
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 12); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+126)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 18}, seatDetails, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Internal divider
	pdf.SetStrokeColor(226, 232, 240)
	pdf.Line(55, splitY+156, 330, splitY+156)

	// Price & Payment Status
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+166)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 12}, "TOTAL AMOUNT & STATUS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	curr := data.Currency
	if curr == "" {
		curr = "IDR"
	}
	amountText := data.TotalAmount
	if amountText == "" {
		amountText = "0"
	}
	fullPrice := fmt.Sprintf("%s %s", curr, amountText)

	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 13); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+180)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 140, H: 20}, fullPrice, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// Payment Status Badge (Green "PAID / CONFIRMED")
	pdf.SetFillColor(220, 252, 231) // green-100
	pdf.RectFromUpperLeftWithStyle(55, splitY+208, 140, 22, "F")
	pdf.SetTextColor(22, 101, 52) // green-800
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(55, splitY+208)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 140, H: 22}, "PAID / CONFIRMED", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	// Page Counter (if multi-seat booking)
	if totalPages > 1 {
		pdf.SetTextColor(100, 116, 139)
		if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil {
			return err
		}
		pdf.SetXY(55, splitY+248)
		pageText := fmt.Sprintf("Ticket %d of %d (Multi-Seat Booking)", pageNum, totalPages)
		_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 14}, pageText, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	}

	// Right Box: QR Code & Verification (Width: 200)
	qrBoxX := 350.0
	pdf.SetFillColor(248, 250, 252)
	pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(qrBoxX, splitY, 200, cardHeight, "FD")

	// Section Label
	pdf.SetTextColor(99, 102, 241)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(qrBoxX+5, splitY+10)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 14}, "SECURE ENTRY PASS", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	// Resolve QR Code PNG Bytes
	qrBytes := data.QRBytes
	if len(qrBytes) == 0 {
		qrPayload := data.QRData
		if qrPayload == "" {
			qrPayload = g.qrGen.GenerateVerificationPayload(data.BookingRef, data.SeatDetails, data.EventTitle)
		}
		generated, err := g.qrGen.Generate(qrPayload)
		if err != nil {
			return fmt.Errorf("failed to generate ticket qr code: %w", err)
		}
		qrBytes = generated
	}

	// Embed QR Image in PDF
	imgHolder, err := gopdf.ImageHolderByBytes(qrBytes)
	if err != nil {
		return fmt.Errorf("failed to create image holder for qr code: %w", err)
	}

	qrImgSize := 140.0
	qrImgX := qrBoxX + (200.0-qrImgSize)/2.0
	qrImgY := splitY + 34.0
	if err := pdf.ImageByHolder(imgHolder, qrImgX, qrImgY, &gopdf.Rect{W: qrImgSize, H: qrImgSize}); err != nil {
		return fmt.Errorf("failed to render qr image into pdf: %w", err)
	}

	// QR Subtitle & Instructions
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 10); err != nil {
		return err
	}
	pdf.SetXY(qrBoxX+5, splitY+185)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 16}, "Scan at entrance gate", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil {
		return err
	}
	pdf.SetXY(qrBoxX+5, splitY+205)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 14}, "Single Entry Verification", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	pdf.SetXY(qrBoxX+5, splitY+220)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 14}, "Do not fold or damage QR code", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	// 5. Perforated Stub Divider
	divY := splitY + cardHeight + 15.0
	pdf.SetStrokeColor(148, 163, 184) // slate-400
	pdf.SetLineWidth(0.8)
	pdf.SetLineType("dashed")
	pdf.Line(35, divY, 560, divY)
	pdf.SetLineType("") // reset to solid

	// 6. Security Advisory & Gate Entry Instructions Section
	advisoryY := divY + 12.0
	advisoryHeight := 130.0
	pdf.SetFillColor(248, 250, 252)
	pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(45, advisoryY, 505, advisoryHeight, "FD")

	pdf.SetTextColor(30, 41, 59)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(55, advisoryY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 485, H: 14}, "GATE ENTRY ADVISORY & SECURITY NOTICE", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	instructions := []string{
		"• Gate Presentation: Present this official e-ticket on your mobile device or printed on clean paper at the turnstile.",
		"• Identity Check: A valid government photo ID matching the Attendee Name may be inspected upon gate arrival.",
		"• Unique Entry Token: The encoded QR code is single-use and invalidates immediately after gate validation.",
		"• Venue Regulations: Outside food/beverages, professional recording gear, and hazardous items are strictly prohibited.",
		"• Security & Fraud: TickId is not liable for duplicated passes. Do not share your QR code on social media.",
	}

	pdf.SetTextColor(71, 85, 105) // slate-600
	if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil {
		return err
	}
	lineY := advisoryY + 26.0
	for _, inst := range instructions {
		pdf.SetXY(55, lineY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: 485, H: 12}, inst, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
		lineY += 15.0
	}

	// 7. Footer Bar
	footerY := advisoryY + advisoryHeight + 12.0
	pdf.SetFillColor(15, 23, 42) // slate-900
	pdf.RectFromUpperLeftWithStyle(30, footerY, 535, 45, "F")

	pdf.SetTextColor(203, 213, 225) // slate-300
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil {
		return err
	}
	pdf.SetXY(45, footerY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 250, H: 14}, "TickId Official Ticketing Platform", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	pdf.SetTextColor(148, 163, 184) // slate-400
	if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil {
		return err
	}
	pdf.SetXY(45, footerY+24)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 250, H: 12}, "support@tickid.com • https://tickid.com", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	pdf.SetXY(300, footerY+16)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 250, H: 14}, "Powered by TickId Microservice Architecture", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

	return nil
}
