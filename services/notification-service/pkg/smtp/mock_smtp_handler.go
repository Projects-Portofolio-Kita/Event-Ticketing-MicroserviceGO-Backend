// Package smtp provides email dispatching clients and an in-memory test server.
//
// File: mock_smtp_handler.go
// Usability: Manages socket I/O loops, greeting banner, and multi-line message buffering in DATA mode.
package smtp

import (
	"bufio"
	"bytes"
	"net"
	"strings"
	"time"
)

func (s *MockSMTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader, writer := bufio.NewReader(conn), bufio.NewWriter(conn)
	writer.WriteString("220 127.0.0.1 TickId Mock SMTP Server Ready\r\n")
	writer.Flush()
	var curFrom string
	var curRcpts []string
	var dataBuf bytes.Buffer
	inData := false
	for {
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if inData {
			if line == ".\r\n" || line == ".\n" {
				inData = false
				raw := dataBuf.Bytes()
				parsed, _ := parseRawEmail(raw, curFrom, curRcpts)
				s.recordMessage(raw, parsed)
				writer.WriteString("250 2.0.0 OK: queued\r\n")
				writer.Flush()
				curFrom = ""; curRcpts = nil; dataBuf.Reset()
			} else {
				if strings.HasPrefix(line, "..") { line = line[1:] }
				dataBuf.WriteString(line)
			}
			continue
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if !handleSMTPCmd(writer, reader, trimmed, strings.ToUpper(trimmed), &curFrom, &curRcpts, &inData) {
			return
		}
	}
}