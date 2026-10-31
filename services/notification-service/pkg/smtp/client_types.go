// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: client_types.go
// Usability: Declares EmailMessage, Attachment, EmailSender interface, and Client configuration structs.
package smtp

import (
	"context"

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
	Host, Username, Password, From, FromName string
	Port                                     int
	UseTLS                                   bool
}

// Client implements EmailSender using github.com/wneessen/go-mail
type Client struct {
	cfg        Config
	mailClient *gomail.Client
}
