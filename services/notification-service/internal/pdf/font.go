package pdf

import (
	_ "embed"
	"fmt"

	"github.com/signintech/gopdf"
)

//go:embed fonts/Go-Regular.ttf
var FontGoRegular []byte

//go:embed fonts/Go-Bold.ttf
var FontGoBold []byte

const (
	// FontFamilyRegular is the registered font family name for regular text
	FontFamilyRegular = "GoRegular"
	// FontFamilyBold is the registered font family name for bold text
	FontFamilyBold = "GoBold"
)

// RegisterEmbeddedFonts loads the embedded TrueType fonts into the provided GoPdf instance.
// This guarantees that PDF generation is 100% self-contained and does not rely on system fonts.
func RegisterEmbeddedFonts(pdf *gopdf.GoPdf) error {
	if pdf == nil {
		return fmt.Errorf("pdf instance cannot be nil")
	}

	if err := pdf.AddTTFFontData(FontFamilyRegular, FontGoRegular); err != nil {
		return fmt.Errorf("failed to load regular font: %w", err)
	}

	if err := pdf.AddTTFFontData(FontFamilyBold, FontGoBold); err != nil {
		return fmt.Errorf("failed to load bold font: %w", err)
	}

	return nil
}
