// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: client_send.go
// Usability: Implements Send method validating message parameters and delivering through SMTP client with context.
package smtp

import (
	"context"
	"errors"
	"strings"

	gomail "github.com/wneessen/go-mail"
)

// Send delivers an EmailMessage over SMTP
func (c *Client) Send(ctx context.Context, msg *EmailMessage) error {
	if msg == nil {
		return errors.New("email message is nil")
	}
	if len(msg.To) == 0 {
		return errors.New("no recipients specified")
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return errors.New("email subject cannot be empty")
	}
	if msg.HTMLBody == "" && msg.TextBody == "" {
		return errors.New("email body (html or text) cannot be empty")
	}
	m := gomail.NewMsg()
	if err := populateGomailMsg(m, c.cfg, msg); err != nil {
		return err
	}
	return c.mailClient.DialAndSendWithContext(ctx, m)
}
