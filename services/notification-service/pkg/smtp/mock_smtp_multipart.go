// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_multipart.go
// Usability: Parses MIME body parts, extracts HTML/text bodies, and decodes binary attachments.
package smtp

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"strings"
)

func parseMIMEContent(contentType string, body []byte, msg *ReceivedMessage) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		if strings.HasPrefix(contentType, "text/html") {
			msg.HTMLBody = string(body)
		} else {
			msg.TextBody = string(body)
		}
		return
	}
	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	for {
		p, err := mr.NextPart()
		if err != nil { break }
		pType, pParams, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		pDisp, pDispParams, _ := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
		pBytes, _ := io.ReadAll(p)
		if p.Header.Get("Content-Transfer-Encoding") == "base64" {
			clean := strings.ReplaceAll(strings.ReplaceAll(string(pBytes), "\r", ""), "\n", "")
			decoded, err := base64.StdEncoding.DecodeString(clean)
			if err == nil { pBytes = decoded }
		}
		filename := pDispParams["filename"]
		if filename == "" { filename = pParams["name"] }
		if pDisp == "attachment" || filename != "" {
			msg.Attachments = append(msg.Attachments, Attachment{Filename: filename, ContentType: pType, Data: pBytes})
		} else if strings.HasPrefix(pType, "text/html") {
			msg.HTMLBody = string(pBytes)
		} else if strings.HasPrefix(pType, "multipart/") {
			parseMIMEContent(p.Header.Get("Content-Type"), pBytes, msg)
		} else {
			msg.TextBody = string(pBytes)
		}
	}
}