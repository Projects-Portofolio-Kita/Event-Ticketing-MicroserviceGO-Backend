// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_types.go
// Usability: Declares ReceivedMessage data model, MockSMTPServer struct, and address inspection methods.
package smtp

import (
	"fmt"
	"net"
	"net/mail"
	"sync"
)

// ReceivedMessage represents a parsed email message captured by MockSMTPServer
type ReceivedMessage struct {
	Raw         []byte
	From        string
	To          []string
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
	Headers     mail.Header
}

// MockSMTPServer is an in-memory RFC-5321 compliant SMTP server for automated testing
type MockSMTPServer struct {
	listener net.Listener
	addr     string
	port     int
	mu       sync.RWMutex
	messages []ReceivedMessage
	rawMsgs  [][]byte
	quit     chan struct{}
	wg       sync.WaitGroup
}

// Host returns the listening IP address
func (s *MockSMTPServer) Host() string { return s.addr }

// Port returns the listening TCP port
func (s *MockSMTPServer) Port() int { return s.port }

// Addr returns the complete host:port string
func (s *MockSMTPServer) Addr() string {
	return fmt.Sprintf("%s:%d", s.addr, s.port)
}
