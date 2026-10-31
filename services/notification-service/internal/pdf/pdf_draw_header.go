// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_draw_header.go
// Usability: Renders the outer container border, branded dark top banner, and booking ref pill.
package pdf

import "github.com/signintech/gopdf"

func renderContainerAndHeader(pdf *gopdf.GoPdf, data TicketPDFData) error {
	pdf.SetStrokeColor(203, 213, 225)
	pdf.SetLineWidth(1)
	pdf.SetFillColor(255, 255, 255)
	pdf.RectFromUpperLeftWithStyle(30, 25, 535, 785, "FD")
	pdf.SetFillColor(30, 41, 59)
	pdf.RectFromUpperLeftWithStyle(30, 25, 535, 70, "F")
	pdf.SetFillColor(99, 102, 241)
	pdf.RectFromUpperLeftWithStyle(30, 25, 535, 4, "F")
	pdf.SetTextColor(255, 255, 255)
	if err := pdf.SetFont(FontFamilyBold, "", 20); err != nil {
		return err
	}
	pdf.SetXY(45, 38)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 200, H: 24}, "TickId", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetTextColor(199, 210, 254)
	if err := pdf.SetFont(FontFamilyRegular, "", 10); err != nil {
		return err
	}
	pdf.SetXY(45, 64)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 200, H: 16}, "OFFICIAL EVENT E-TICKET", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetFillColor(49, 46, 129)
	pdf.RectFromUpperLeftWithStyle(355, 40, 195, 36, "F")
	pdf.SetTextColor(224, 231, 255)
	if err := pdf.SetFont(FontFamilyBold, "", 8); err != nil {
		return err
	}
	pdf.SetXY(362, 44)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 180, H: 12}, "BOOKING REFERENCE", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	ref := data.BookingRef
	if ref == "" {
		ref = "N/A"
	}
	pdf.SetTextColor(255, 255, 255)
	if err := pdf.SetFont(FontFamilyBold, "", 11); err != nil {
		return err
	}
	pdf.SetXY(362, 58)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 180, H: 14}, ref, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	return nil
}