// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: client_msg.go
// Usability: Populates go-mail Message attributes including sender headers, recipients, body alternatives, and binary attachments.
package smtp

import (
	"bytes"
	"fmt"

	gomail "github.com/wneessen/go-mail"
)

func populateGomailMsg(m *gomail.Msg, cfg Config, msg *EmailMessage) error {
	if cfg.FromName != "" {
		if err := m.FromFormat(cfg.FromName, cfg.From); err != nil && m.From(cfg.From) != nil {
			return fmt.Errorf("invalid from address: %w", err)
		}
	} else if err := m.From(cfg.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if err := m.To(msg.To...); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	m.Subject(msg.Subject)
	if msg.TextBody != "" && msg.HTMLBody != "" {
		m.SetBodyString(gomail.TypeTextPlain, msg.TextBody)
		m.AddAlternativeString(gomail.TypeTextHTML, msg.HTMLBody)
	} else if msg.HTMLBody != "" {
		m.SetBodyString(gomail.TypeTextHTML, msg.HTMLBody)
	} else {
		m.SetBodyString(gomail.TypeTextPlain, msg.TextBody)
	}
	for _, att := range msg.Attachments {
		if att.Filename == "" {
			continue
		}
		var attachOpts []gomail.FileOption
		if att.ContentType != "" {
			attachOpts = append(attachOpts, gomail.WithFileContentType(gomail.ContentType(att.ContentType)))
		}
		if err := m.AttachReader(att.Filename, bytes.NewReader(att.Data), attachOpts...); err != nil {
			return fmt.Errorf("failed to attach file %s: %w", att.Filename, err)
		}
	}
	return nil
}
