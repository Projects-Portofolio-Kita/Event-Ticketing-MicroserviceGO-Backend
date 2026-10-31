// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: client.go
// Usability: Initializes SMTP clients with host configurations, authentication, and TLS options.
package smtp

import (
	"crypto/tls"
	"errors"
	"fmt"

	gomail "github.com/wneessen/go-mail"
)

// NewClient creates a new SMTP client with explicit positional parameters
func NewClient(host string, port int, user, pass, from string, useTLS bool) (*Client, error) {
	return NewClientWithConfig(Config{
		Host: host, Port: port, Username: user, Password: pass, From: from, FromName: "TickId Official", UseTLS: useTLS,
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
	opts := []gomail.Option{gomail.WithPort(cfg.Port)}
	if cfg.UseTLS {
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory), gomail.WithTLSConfig(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}))
	} else {
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	}
	if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthPlain), gomail.WithUsername(cfg.Username), gomail.WithPassword(cfg.Password))
	}
	mClient, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize smtp client: %w", err)
	}
	return &Client{cfg: cfg, mailClient: mClient}, nil
}
