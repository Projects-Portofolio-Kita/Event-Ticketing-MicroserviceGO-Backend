// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_draw_attendee.go
// Usability: Renders attendee name, ticket tier badge, seat details, and total amount.
package pdf

import (
	"strings"

	"github.com/signintech/gopdf"
)

func renderAttendeeSection(pdf *gopdf.GoPdf, data TicketPDFData, splitY, cardH float64, pageNum, totalPages int) error {
	pdf.SetFillColor(248, 250, 252); pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(45, splitY, 295, cardH, "FD")
	pdf.SetTextColor(99, 102, 241)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(55, splitY+10)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 14}, "ATTENDEE & TICKET SPECIFICATIONS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	name := data.AttendeeName
	if name == "" { name = "Valued Guest" }
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 13); err != nil { return err }
	pdf.SetXY(55, splitY+44)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 18}, name, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	tier := strings.ToUpper(data.TierName)
	if tier == "" { tier = "GENERAL ADMISSION" }
	pdf.SetFillColor(224, 231, 255); pdf.RectFromUpperLeftWithStyle(55, splitY+82, 160, 22, "F")
	pdf.SetTextColor(67, 56, 202)
	if err := pdf.SetFont(FontFamilyBold, "", 10); err != nil { return err }
	pdf.SetXY(60, splitY+82)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 150, H: 22}, tier, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	seat := data.SeatDetails
	if seat == "" { seat = "Free Seating / General Entry" }
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 12); err != nil { return err }
	pdf.SetXY(55, splitY+126)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 18}, seat, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	return renderPaymentDetails(pdf, data, splitY, pageNum, totalPages)
}