// Package template provides HTML and text rendering engines for email notifications.
//
// File: engine_helpers.go
// Usability: Provides helper parser to compile dual HTML and plain-text template pairs.
package template

import (
	htmltmpl "html/template"
	texttmpl "text/template"
)

func parseHTMLAndText(hName, hStr, tName, tStr string, fMap htmltmpl.FuncMap) (*htmltmpl.Template, *texttmpl.Template, error) {
	hTmpl, err := htmltmpl.New(hName).Funcs(fMap).Parse(hStr)
	if err != nil {
		return nil, nil, err
	}
	tTmpl, err := texttmpl.New(tName).Parse(tStr)
	if err != nil {
		return nil, nil, err
	}
	return hTmpl, tTmpl, nil
}
