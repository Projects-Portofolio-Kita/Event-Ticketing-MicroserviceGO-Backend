// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_find.go
// Usability: Provides search routines for captured messages by recipient address or subject substring, and reset.
package smtp

import "strings"

// Reset clears all captured messages and raw buffers
func (s *MockSMTPServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = s.messages[:0]
	s.rawMsgs = s.rawMsgs[:0]
}

// FindMessagesTo finds all received messages with the specified recipient
func (s *MockSMTPServer) FindMessagesTo(to string) []ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []ReceivedMessage
	for _, msg := range s.messages {
		for _, rcpt := range msg.To {
			if strings.EqualFold(rcpt, to) {
				results = append(results, msg)
				break
			}
		}
	}
	return results
}

// FindMessageBySubject searches for the first message whose subject contains substr
func (s *MockSMTPServer) FindMessageBySubject(substr string) *ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, msg := range s.messages {
		if strings.Contains(strings.ToLower(msg.Subject), strings.ToLower(substr)) {
			cpy := msg
			return &cpy
		}
	}
	return nil
}