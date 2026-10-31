// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_draw_payment.go
// Usability: Renders payment badge, currency amount, and multi-seat page counter.
package pdf

import (
	"fmt"

	"github.com/signintech/gopdf"
)

func renderPaymentDetails(pdf *gopdf.GoPdf, data TicketPDFData, splitY float64, pageNum, totalPages int) error {
	pdf.SetStrokeColor(226, 232, 240)
	pdf.Line(55, splitY+156, 330, splitY+156)
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil { return err }
	pdf.SetXY(55, splitY+166)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 12}, "TOTAL AMOUNT & STATUS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	curr := data.Currency
	if curr == "" { curr = "IDR" }
	amt := data.TotalAmount
	if amt == "" { amt = "0" }
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 13); err != nil { return err }
	pdf.SetXY(55, splitY+180)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 140, H: 20}, fmt.Sprintf("%s %s", curr, amt), gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetFillColor(220, 252, 231)
	pdf.RectFromUpperLeftWithStyle(55, splitY+208, 140, 22, "F")
	pdf.SetTextColor(22, 101, 52)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(55, splitY+208)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 140, H: 22}, "PAID / CONFIRMED", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	if totalPages > 1 {
		pdf.SetTextColor(100, 116, 139)
		if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil { return err }
		pdf.SetXY(55, splitY+248)
		_ = pdf.CellWithOption(&gopdf.Rect{W: 275, H: 14}, fmt.Sprintf("Ticket %d of %d (Multi-Seat Booking)", pageNum, totalPages), gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	}
	return nil
}