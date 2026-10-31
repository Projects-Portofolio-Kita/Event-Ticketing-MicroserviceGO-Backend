// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp.go
// Usability: Sets up ephemeral mock TCP listener, serves connections, and manages shutdown lifecycle.
package smtp

import (
	"fmt"
	"net"
)

// NewMockSMTPServer starts an in-memory SMTP listener on an ephemeral port (127.0.0.1:0)
func NewMockSMTPServer() (*MockSMTPServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind mock smtp listener: %w", err)
	}
	tcpAddr := ln.Addr().(*net.TCPAddr)
	server := &MockSMTPServer{
		listener: ln, addr: tcpAddr.IP.String(), port: tcpAddr.Port,
		messages: make([]ReceivedMessage, 0), rawMsgs: make([][]byte, 0), quit: make(chan struct{}),
	}
	server.wg.Add(1); go server.serve()
	return server, nil
}

func (s *MockSMTPServer) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				continue
			}
		}
		s.wg.Add(1)
		go func(c net.Conn) { defer s.wg.Done(); s.handleConnection(c) }(conn)
	}
}

// Close gracefully closes the listener and terminates running connection handlers
func (s *MockSMTPServer) Close() error {
	close(s.quit); err := s.listener.Close()
	s.wg.Wait()
	return err
}