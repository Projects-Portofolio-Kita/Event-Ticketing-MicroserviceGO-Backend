// Package template provides HTML and text rendering engines for email notifications.
//
// File: engine.go
// Usability: Defines TemplateEngine struct, template URL escaping helpers, and template compiler constructor.
package template

import (
	htmltmpl "html/template"
	"strings"
	texttmpl "text/template"
)

// TemplateEngine is the concrete implementation of Engine.
type TemplateEngine struct {
	orderConfirmHTML *htmltmpl.Template
	orderConfirmText *texttmpl.Template
	paymentFailHTML  *htmltmpl.Template
	paymentFailText  *texttmpl.Template
	eventNoticeHTML  *htmltmpl.Template
	eventNoticeText  *texttmpl.Template
}

func safeQR(s string) htmltmpl.URL {
	if s == "" { return "" }
	if !strings.HasPrefix(s, "data:image") {
		return htmltmpl.URL("data:image/png;base64," + s)
	}
	return htmltmpl.URL(s)
}

func safeURL(s string) htmltmpl.URL { return htmltmpl.URL(s) }

// NewEngine creates and parses all HTML and plain-text templates.
func NewEngine() (*TemplateEngine, error) {
	fMap := htmltmpl.FuncMap{"safeQR": safeQR, "safeURL": safeURL}
	ocH, ocT, err := parseHTMLAndText("order_confirmation_html", orderConfirmationHTMLTemplate, "order_confirmation_text", orderConfirmationTextTemplate, fMap)
	if err != nil { return nil, err }
	pfH, pfT, err := parseHTMLAndText("payment_failed_html", paymentFailedHTMLTemplate, "payment_failed_text", paymentFailedTextTemplate, fMap)
	if err != nil { return nil, err }
	enH, enT, err := parseHTMLAndText("event_notice_html", eventNoticeHTMLTemplate, "event_notice_text", eventNoticeTextTemplate, fMap)
	if err != nil { return nil, err }
	return &TemplateEngine{orderConfirmHTML: ocH, orderConfirmText: ocT, paymentFailHTML: pfH, paymentFailText: pfT, eventNoticeHTML: enH, eventNoticeText: enT}, nil
}