// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_parser.go
// Usability: Parses raw RFC-822 email payload, extracts MIME headers, and normalizes envelope metadata.
package smtp

import (
	"bytes"
	"io"
	"net/mail"
	"strings"
)

func extractEmailAddress(s string) string {
	s = strings.TrimSpace(s)
	if start := strings.Index(s, "<"); start != -1 {
		if end := strings.Index(s[start:], ">"); end != -1 {
			return s[start+1 : start+end]
		}
	}
	return strings.Trim(s, "<>")
}

func parseRawEmail(raw []byte, envFrom string, envTo []string) (ReceivedMessage, error) {
	msg := ReceivedMessage{Raw: raw, From: envFrom, To: envTo}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return msg, err
	}
	msg.Headers = m.Header
	msg.Subject = m.Header.Get("Subject")
	if envFrom == "" {
		msg.From = extractEmailAddress(m.Header.Get("From"))
	}
	if len(envTo) == 0 && m.Header.Get("To") != "" {
		for _, addr := range strings.Split(m.Header.Get("To"), ",") {
			if clean := extractEmailAddress(addr); clean != "" {
				msg.To = append(msg.To, clean)
			}
		}
	}
	contentType := m.Header.Get("Content-Type")
	bodyBytes, _ := io.ReadAll(m.Body)
	parseMIMEContent(contentType, bodyBytes, &msg)
	return msg, nil
}