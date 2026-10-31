// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_queries.go
// Usability: Provides thread-safe storage, count, and retrieval operations for mock messages.
package smtp

func (s *MockSMTPServer) recordMessage(raw []byte, parsed ReceivedMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, parsed)
	s.rawMsgs = append(s.rawMsgs, raw)
}

// GetReceivedMessages returns copies of all parsed messages received by the server
func (s *MockSMTPServer) GetReceivedMessages() []ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ReceivedMessage, len(s.messages))
	copy(out, s.messages)
	return out
}

// GetRawMessages returns copies of all raw message byte slices
func (s *MockSMTPServer) GetRawMessages() [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([][]byte, len(s.rawMsgs))
	copy(out, s.rawMsgs)
	return out
}

// MessageCount returns the number of captured messages
func (s *MockSMTPServer) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

// LastMessage returns the most recent message received, or nil if none
func (s *MockSMTPServer) LastMessage() *ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.messages) == 0 {
		return nil
	}
	cpy := s.messages[len(s.messages)-1]
	return &cpy
}