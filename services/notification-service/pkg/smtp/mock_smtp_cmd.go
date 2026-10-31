// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_cmd.go
// Usability: Dispatches RFC-5321 SMTP command verbs (EHLO, MAIL, RCPT, AUTH, DATA, RSET, QUIT).
package smtp

import (
	"bufio"
	"strings"
)

func handleSMTPCmd(w *bufio.Writer, r *bufio.Reader, trimmed, cmd string, from *string, rcpts *[]string, inData *bool) bool {
	switch {
	case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
		w.WriteString("250-127.0.0.1\r\n250-PIPELINING\r\n250-SIZE 35882577\r\n250-8BITMIME\r\n250-AUTH PLAIN LOGIN\r\n250 OK\r\n")
	case strings.HasPrefix(cmd, "AUTH PLAIN"):
		parts := strings.SplitN(trimmed, " ", 3)
		if len(parts) >= 3 && parts[2] != "" {
			w.WriteString("235 2.7.0 Authentication successful\r\n")
		} else {
			w.WriteString("334 \r\n"); w.Flush()
			if _, err := r.ReadString('\n'); err != nil { return false }
			w.WriteString("235 2.7.0 Authentication successful\r\n")
		}
	case strings.HasPrefix(cmd, "AUTH LOGIN"):
		w.WriteString("334 VXNlcm5hbWU6\r\n"); w.Flush()
		if _, err := r.ReadString('\n'); err != nil { return false }
		w.WriteString("334 UGFzc3dvcmQ6\r\n"); w.Flush()
		if _, err := r.ReadString('\n'); err != nil { return false }
		w.WriteString("235 2.7.0 Authentication successful\r\n")
	case strings.HasPrefix(cmd, "MAIL FROM:"):
		*from = extractEmailAddress(trimmed[10:])
		w.WriteString("250 2.1.0 Sender OK\r\n")
	case strings.HasPrefix(cmd, "RCPT TO:"):
		if to := extractEmailAddress(trimmed[8:]); to != "" { *rcpts = append(*rcpts, to) }
		w.WriteString("250 2.1.5 Recipient OK\r\n")
	case cmd == "DATA":
		*inData = true; w.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
	case cmd == "RSET":
		*from, *rcpts = "", nil; w.WriteString("250 2.0.0 OK\r\n")
	case cmd == "NOOP":
		w.WriteString("250 2.0.0 OK\r\n")
	case cmd == "QUIT":
		w.WriteString("221 2.0.0 Service closing transmission channel\r\n"); w.Flush()
		return false
	default:
		w.WriteString("500 5.5.1 Command unrecognized\r\n")
	}
	w.Flush(); return true
}