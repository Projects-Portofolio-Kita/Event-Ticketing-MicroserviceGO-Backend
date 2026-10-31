// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_draw_qr.go
// Usability: Resolves QR payload, generates image raster, and draws the QR pass box into PDF.
package pdf

import (
	"fmt"

	"github.com/signintech/gopdf"
)

func (g *TicketPDFGenerator) renderQRSection(pdf *gopdf.GoPdf, data TicketPDFData, splitY, cardH float64) error {
	qrBoxX := 350.0
	pdf.SetFillColor(248, 250, 252); pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(qrBoxX, splitY, 200, cardH, "FD")
	pdf.SetTextColor(99, 102, 241)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(qrBoxX+5, splitY+10)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 14}, "SECURE ENTRY PASS", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	qrBytes := data.QRBytes
	if len(qrBytes) == 0 {
		qrPayload := data.QRData
		if qrPayload == "" {
			qrPayload = g.qrGen.GenerateVerificationPayload(data.BookingRef, data.SeatDetails, data.EventTitle)
		}
		gen, err := g.qrGen.Generate(qrPayload)
		if err != nil { return fmt.Errorf("failed to generate ticket qr code: %w", err) }
		qrBytes = gen
	}
	imgHolder, err := gopdf.ImageHolderByBytes(qrBytes)
	if err != nil { return fmt.Errorf("failed to create image holder for qr code: %w", err) }
	qrImgSize := 140.0
	qrImgX := qrBoxX + (200.0-qrImgSize)/2.0
	if err := pdf.ImageByHolder(imgHolder, qrImgX, splitY+34.0, &gopdf.Rect{W: qrImgSize, H: qrImgSize}); err != nil {
		return fmt.Errorf("failed to render qr image into pdf: %w", err)
	}
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 10); err != nil { return err }
	pdf.SetXY(qrBoxX+5, splitY+185)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 16}, "Scan at entrance gate", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil { return err }
	pdf.SetXY(qrBoxX+5, splitY+205)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 14}, "Single Entry Verification", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(qrBoxX+5, splitY+220)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 190, H: 14}, "Do not fold or damage QR code", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	return nil
}