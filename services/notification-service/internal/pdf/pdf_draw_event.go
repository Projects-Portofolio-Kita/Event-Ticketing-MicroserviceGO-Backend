// Package pdf provides e-ticket generation with branded layout, typography, and QR codes.
//
// File: pdf_draw_event.go
// Usability: Renders event details card containing event title, date, time, and venue information.
package pdf

import "github.com/signintech/gopdf"

func renderEventDetails(pdf *gopdf.GoPdf, data TicketPDFData) error {
	eventY := 105.0
	pdf.SetFillColor(248, 250, 252); pdf.SetStrokeColor(226, 232, 240)
	pdf.RectFromUpperLeftWithStyle(45, eventY, 505, 125, "FD")
	pdf.SetTextColor(99, 102, 241)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(55, eventY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 485, H: 14}, "EVENT DETAILS", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	title := data.EventTitle
	if title == "" { title = "Untitled Event" }
	pdf.SetTextColor(15, 23, 42)
	if err := pdf.SetFont(FontFamilyBold, "", 16); err != nil { return err }
	pdf.SetXY(55, eventY+24)
	_ = pdf.MultiCell(&gopdf.Rect{W: 485, H: 20}, title)
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(55, eventY+68)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 80, H: 14}, "DATE & TIME:", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	dt := data.EventDate
	if dt == "" { dt = "TBA" }
	pdf.SetTextColor(30, 41, 59)
	if err := pdf.SetFont(FontFamilyRegular, "", 10); err != nil { return err }
	pdf.SetXY(135, eventY+68)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 405, H: 14}, dt, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetTextColor(100, 116, 139)
	if err := pdf.SetFont(FontFamilyBold, "", 9); err != nil { return err }
	pdf.SetXY(55, eventY+86)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 80, H: 14}, "VENUE:", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	venue := data.VenueName
	if data.VenueAddress != "" {
		if venue != "" { venue += " — " + data.VenueAddress } else { venue = data.VenueAddress }
	}
	if venue == "" { venue = "Venue location to be announced" }
	pdf.SetTextColor(30, 41, 59)
	if err := pdf.SetFont(FontFamilyRegular, "", 10); err != nil { return err }
	pdf.SetXY(135, eventY+86)
	_ = pdf.MultiCell(&gopdf.Rect{W: 405, H: 15}, venue)
	return nil
}