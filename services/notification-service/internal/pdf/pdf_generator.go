// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_generator.go
// Usability: Implements TicketPDFGenerator constructor and single ticket delegation.
package pdf

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
	return &TicketPDFGenerator{qrGen: qrGen}
}

// GenerateTicket produces a single-ticket PDF document in-memory.
func (g *TicketPDFGenerator) GenerateTicket(data TicketPDFData) ([]byte, error) {
	return g.GenerateMultiTicket([]TicketPDFData{data})
}