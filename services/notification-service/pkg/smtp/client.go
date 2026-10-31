package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	gomail "github.com/wneessen/go-mail"
)

// EmailMessage contains all attributes required to deliver an email
type EmailMessage struct {
	To          []string
	Subject     string
	HTMLBody    string
	TextBody    string
	Attachments []Attachment
}

// Attachment represents an in-memory email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// EmailSender defines the interface for delivering email messages
type EmailSender interface {
	Send(ctx context.Context, msg *EmailMessage) error
}

// Config holds SMTP client configuration parameters
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

// Client implements EmailSender using github.com/wneessen/go-mail
type Client struct {
	cfg        Config
	mailClient *gomail.Client
}

// NewClient creates a new SMTP client with explicit positional parameters
func NewClient(host string, port int, user, pass, from string, useTLS bool) (*Client, error) {
	return NewClientWithConfig(Config{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		From:     from,
		FromName: "TickId Official",
		UseTLS:   useTLS,
	})
}

// NewClientWithConfig creates a new SMTP client using Config
func NewClientWithConfig(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, errors.New("smtp host is required")
	}
	if cfg.Port <= 0 {
		cfg.Port = 1025
	}
	if cfg.From == "" {
		cfg.From = "noreply@tickid.com"
	}

	opts := []gomail.Option{
		gomail.WithPort(cfg.Port),
	}

	if cfg.UseTLS {
		opts = append(opts,
			gomail.WithTLSPolicy(gomail.TLSMandatory),
			gomail.WithTLSConfig(&tls.Config{
				ServerName: cfg.Host,
				MinVersion: tls.VersionTLS12,
			}),
		)
	} else {
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	}

	if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(cfg.Username),
			gomail.WithPassword(cfg.Password),
		)
	}

	mClient, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize smtp client: %w", err)
	}

	return &Client{
		cfg:        cfg,
		mailClient: mClient,
	}, nil
}

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
	if c.cfg.FromName != "" {
		if err := m.FromFormat(c.cfg.FromName, c.cfg.From); err != nil {
			if err := m.From(c.cfg.From); err != nil {
				return fmt.Errorf("invalid from address: %w", err)
			}
		}
	} else {
		if err := m.From(c.cfg.From); err != nil {
			return fmt.Errorf("invalid from address: %w", err)
		}
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

	return c.mailClient.DialAndSendWithContext(ctx, m)
}
