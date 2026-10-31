// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_draw_advisory.go
// Usability: Renders gate entry instructions, security terms, and platform branding footer.
package pdf

import "github.com/signintech/gopdf"

func renderAdvisoryAndFooter(pdf *gopdf.GoPdf, divY float64) error {
	pdf.SetStrokeColor(148, 163, 184); pdf.SetLineWidth(0.8); pdf.SetLineType("dashed")
	pdf.Line(35, divY, 560, divY); pdf.SetLineType("")
	advisoryY, advH := divY+12.0, 130.0
	pdf.SetFillColor(248, 250, 252); pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(45, advisoryY, 505, advH, "FD")
	pdf.SetTextColor(30, 41, 59)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(55, advisoryY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 485, H: 14}, "GATE ENTRY ADVISORY & SECURITY NOTICE", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	instructions := []string{
		"• Gate Presentation: Present this official e-ticket on your mobile device or printed on clean paper at the turnstile.",
		"• Identity Check: A valid government photo ID matching the Attendee Name may be inspected upon gate arrival.",
		"• Unique Entry Token: The encoded QR code is single-use and invalidates immediately after gate validation.",
		"• Venue Regulations: Outside food/beverages, professional recording gear, and hazardous items are strictly prohibited.",
		"• Security & Fraud: TickId is not liable for duplicated passes. Do not share your QR code on social media.",
	}
	pdf.SetTextColor(71, 85, 105)
	if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil { return err }
	lineY := advisoryY + 26.0
	for _, inst := range instructions {
		pdf.SetXY(55, lineY); _ = pdf.CellWithOption(&gopdf.Rect{W: 485, H: 12}, inst, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle}); lineY += 15.0
	}
	return renderFooter(pdf, advisoryY+advH+12.0)
}

func renderFooter(pdf *gopdf.GoPdf, footerY float64) error {
	pdf.SetFillColor(15, 23, 42); pdf.RectFromUpperLeftWithStyle(30, footerY, 535, 45, "F")
	pdf.SetTextColor(203, 213, 225)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(45, footerY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 250, H: 14}, "TickId Official Ticketing Platform", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetTextColor(148, 163, 184)
	if err := pdf.SetFont(FontFamilyRegular, "", 8); err != nil { return err }
	pdf.SetXY(45, footerY+24)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 250, H: 12}, "support@tickid.com • https://tickid.com", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(300, footerY+16)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 250, H: 14}, "Powered by TickId Microservice Architecture", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
	return nil
}