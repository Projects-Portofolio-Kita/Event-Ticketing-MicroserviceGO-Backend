// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_page.go
// Usability: Coordinates rendering pipeline for header, details, attendee, QR code, and advisory.
package pdf

import "github.com/signintech/gopdf"

func (g *TicketPDFGenerator) renderTicketPage(pdf *gopdf.GoPdf, data TicketPDFData, pageNum, totalPages int) error {
	if err := renderContainerAndHeader(pdf, data); err != nil {
		return err
	}
	if err := renderEventDetails(pdf, data); err != nil {
		return err
	}
	splitY := 240.0
	cardHeight := 280.0
	if err := renderAttendeeSection(pdf, data, splitY, cardHeight, pageNum, totalPages); err != nil {
		return err
	}
	if err := g.renderQRSection(pdf, data, splitY, cardHeight); err != nil {
		return err
	}
	return renderAdvisoryAndFooter(pdf, splitY+cardHeight+15.0)
}