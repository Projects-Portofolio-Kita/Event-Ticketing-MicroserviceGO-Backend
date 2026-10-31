// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_multi.go
// Usability: Orchestrates multi-page document creation, font initialization, and byte rendering.
package pdf

import (
	"fmt"

	"github.com/signintech/gopdf"
)

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