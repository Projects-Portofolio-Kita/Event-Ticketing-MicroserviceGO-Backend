package smtp

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"sync"
	"time"
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

// NewMockSMTPServer starts an in-memory SMTP listener on an ephemeral port (127.0.0.1:0)
func NewMockSMTPServer() (*MockSMTPServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind mock smtp listener: %w", err)
	}

	tcpAddr := ln.Addr().(*net.TCPAddr)
	server := &MockSMTPServer{
		listener: ln,
		addr:     tcpAddr.IP.String(),
		port:     tcpAddr.Port,
		messages: make([]ReceivedMessage, 0),
		rawMsgs:  make([][]byte, 0),
		quit:     make(chan struct{}),
	}

	server.wg.Add(1)
	go server.serve()

	return server, nil
}

// Host returns the listening IP address
func (s *MockSMTPServer) Host() string {
	return s.addr
}

// Port returns the listening TCP port
func (s *MockSMTPServer) Port() int {
	return s.port
}

// Addr returns the complete host:port string
func (s *MockSMTPServer) Addr() string {
	return fmt.Sprintf("%s:%d", s.addr, s.port)
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
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleConnection(c)
		}(conn)
	}
}

func (s *MockSMTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send initial SMTP greeting
	writer.WriteString("220 127.0.0.1 TickId Mock SMTP Server Ready\r\n")
	writer.Flush()

	var (
		currentFrom  string
		currentRcpts []string
		dataBuf      bytes.Buffer
		inDataMode   bool
	)

	for {
		// Set read deadline to avoid lingering connections
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		if inDataMode {
			if line == ".\r\n" || line == ".\n" {
				inDataMode = false
				raw := dataBuf.Bytes()

				// Parse message
				parsed, _ := parseRawEmail(raw, currentFrom, currentRcpts)

				s.mu.Lock()
				s.rawMsgs = append(s.rawMsgs, append([]byte(nil), raw...))
				s.messages = append(s.messages, parsed)
				s.mu.Unlock()

				writer.WriteString("250 2.0.0 OK: queued\r\n")
				writer.Flush()

				// Reset transaction state
				currentFrom = ""
				currentRcpts = nil
				dataBuf.Reset()
			} else {
				// Strip leading dot if dot-stuffed
				cleanLine := line
				if strings.HasPrefix(cleanLine, "..") {
					cleanLine = cleanLine[1:]
				}
				dataBuf.WriteString(cleanLine)
			}
			continue
		}

		trimmed := strings.TrimRight(line, "\r\n")
		cmdUpper := strings.ToUpper(trimmed)

		switch {
		case strings.HasPrefix(cmdUpper, "EHLO"), strings.HasPrefix(cmdUpper, "HELO"):
			writer.WriteString("250-127.0.0.1\r\n250-PIPELINING\r\n250-SIZE 35882577\r\n250-8BITMIME\r\n250-AUTH PLAIN LOGIN\r\n250 OK\r\n")

		case strings.HasPrefix(cmdUpper, "AUTH PLAIN"):
			parts := strings.SplitN(trimmed, " ", 3)
			if len(parts) >= 3 && parts[2] != "" {
				writer.WriteString("235 2.7.0 Authentication successful\r\n")
			} else {
				writer.WriteString("334 \r\n")
				writer.Flush()
				_, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				writer.WriteString("235 2.7.0 Authentication successful\r\n")
			}

		case strings.HasPrefix(cmdUpper, "AUTH LOGIN"):
			writer.WriteString("334 VXNlcm5hbWU6\r\n") // Username:
			writer.Flush()
			_, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			writer.WriteString("334 UGFzc3dvcmQ6\r\n") // Password:
			writer.Flush()
			_, err = reader.ReadString('\n')
			if err != nil {
				return
			}
			writer.WriteString("235 2.7.0 Authentication successful\r\n")

		case strings.HasPrefix(cmdUpper, "MAIL FROM:"):
			fromAddr := extractEmailAddress(trimmed[10:])
			currentFrom = fromAddr
			writer.WriteString("250 2.1.0 Sender OK\r\n")

		case strings.HasPrefix(cmdUpper, "RCPT TO:"):
			toAddr := extractEmailAddress(trimmed[8:])
			if toAddr != "" {
				currentRcpts = append(currentRcpts, toAddr)
			}
			writer.WriteString("250 2.1.5 Recipient OK\r\n")

		case cmdUpper == "DATA":
			inDataMode = true
			dataBuf.Reset()
			writer.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")

		case cmdUpper == "RSET":
			currentFrom = ""
			currentRcpts = nil
			dataBuf.Reset()
			writer.WriteString("250 2.0.0 OK\r\n")

		case cmdUpper == "NOOP":
			writer.WriteString("250 2.0.0 OK\r\n")

		case cmdUpper == "QUIT":
			writer.WriteString("221 2.0.0 Service closing transmission channel\r\n")
			writer.Flush()
			return

		default:
			writer.WriteString("500 5.5.1 Command unrecognized\r\n")
		}

		writer.Flush()
	}
}

func extractEmailAddress(raw string) string {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "<")
	end := strings.Index(raw, ">")
	if start != -1 && end != -1 && end > start {
		return raw[start+1 : end]
	}
	// Fallback to first word if not enclosed in brackets
	fields := strings.Fields(raw)
	if len(fields) > 0 {
		return strings.Trim(fields[0], "<>")
	}
	return raw
}

func parseRawEmail(raw []byte, envelopeFrom string, envelopeTo []string) (ReceivedMessage, error) {
	parsedMsg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return ReceivedMessage{
			Raw:  raw,
			From: envelopeFrom,
			To:   envelopeTo,
		}, err
	}

	rec := ReceivedMessage{
		Raw:      raw,
		From:     parsedMsg.Header.Get("From"),
		To:       envelopeTo,
		Subject:  parsedMsg.Header.Get("Subject"),
		Headers:  parsedMsg.Header,
		Attachments: make([]Attachment, 0),
	}

	if rec.From == "" {
		rec.From = envelopeFrom
	}

	contentType := parsedMsg.Header.Get("Content-Type")
	if contentType == "" {
		body, _ := io.ReadAll(parsedMsg.Body)
		rec.TextBody = string(body)
		return rec, nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		body, _ := io.ReadAll(parsedMsg.Body)
		if strings.Contains(mediaType, "text/html") {
			rec.HTMLBody = string(body)
		} else {
			rec.TextBody = string(body)
		}
		return rec, nil
	}

	parseMultipartParts(parsedMsg.Body, params["boundary"], &rec)
	return rec, nil
}

func parseMultipartParts(r io.Reader, boundary string, rec *ReceivedMessage) {
	if boundary == "" {
		return
	}
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}

		partType := part.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(partType)
		filename := part.FileName()

		var partReader io.Reader = part
		cte := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
		if cte == "base64" {
			partReader = base64.NewDecoder(base64.StdEncoding, part)
		} else if cte == "quoted-printable" {
			partReader = quotedprintable.NewReader(part)
		}

		if strings.HasPrefix(mediaType, "multipart/") {
			parseMultipartParts(partReader, params["boundary"], rec)
			continue
		}

		if filename != "" {
			data, _ := io.ReadAll(partReader)
			rec.Attachments = append(rec.Attachments, Attachment{
				Filename:    filename,
				ContentType: mediaType,
				Data:        data,
			})
			continue
		}

		disp, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if disp == "attachment" || dispParams["filename"] != "" {
			attName := dispParams["filename"]
			if attName == "" {
				attName = "attachment"
			}
			data, _ := io.ReadAll(partReader)
			rec.Attachments = append(rec.Attachments, Attachment{
				Filename:    attName,
				ContentType: mediaType,
				Data:        data,
			})
			continue
		}

		if strings.Contains(mediaType, "text/html") {
			body, _ := io.ReadAll(partReader)
			rec.HTMLBody = string(body)
		} else if strings.Contains(mediaType, "text/plain") {
			body, _ := io.ReadAll(partReader)
			rec.TextBody = string(body)
		}
	}
}

// GetReceivedMessages returns a thread-safe copy of all parsed received messages
func (s *MockSMTPServer) GetReceivedMessages() []ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ReceivedMessage, len(s.messages))
	copy(res, s.messages)
	return res
}

// GetRawMessages returns a thread-safe copy of raw message byte buffers
func (s *MockSMTPServer) GetRawMessages() [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([][]byte, len(s.rawMsgs))
	for i, msg := range s.rawMsgs {
		res[i] = append([]byte(nil), msg...)
	}
	return res
}

// MessageCount returns the number of captured messages
func (s *MockSMTPServer) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

// LastMessage returns the most recently captured message, or nil if none
func (s *MockSMTPServer) LastMessage() *ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.messages) == 0 {
		return nil
	}
	msg := s.messages[len(s.messages)-1]
	return &msg
}

// FindMessagesTo finds all captured messages addressed to the given recipient
func (s *MockSMTPServer) FindMessagesTo(recipient string) []ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matches []ReceivedMessage
	for _, m := range s.messages {
		for _, rcpt := range m.To {
			if strings.EqualFold(rcpt, recipient) {
				matches = append(matches, m)
				break
			}
		}
	}
	return matches
}

// FindMessageBySubject finds the first message matching the specified subject
func (s *MockSMTPServer) FindMessageBySubject(subject string) *ReceivedMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.messages {
		if strings.Contains(m.Subject, subject) {
			msg := m
			return &msg
		}
	}
	return nil
}

// Reset clears all recorded messages
func (s *MockSMTPServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = make([]ReceivedMessage, 0)
	s.rawMsgs = make([][]byte, 0)
}

// Close gracefully closes the listener and terminates running connection handlers
func (s *MockSMTPServer) Close() error {
	close(s.quit)
	err := s.listener.Close()
	s.wg.Wait()
	return err
}
