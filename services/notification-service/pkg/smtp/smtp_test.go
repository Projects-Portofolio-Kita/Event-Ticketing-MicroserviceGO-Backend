package smtp_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/smtp"
)

func TestClient_Initialization(t *testing.T) {
	// Test empty host error
	_, err := smtp.NewClient("", 1025, "", "", "noreply@tickid.com", false)
	if err == nil {
		t.Fatal("expected error when host is empty")
	}

	// Test default parameters
	c, err := smtp.NewClient("127.0.0.1", 0, "", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestMockSMTPServer_BasicPlainText(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "sender@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &smtp.EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test Plain Email",
		TextBody: "Hello, this is a plain text test email.",
	}

	err = client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("failed to send plain email: %v", err)
	}

	if mockServer.MessageCount() != 1 {
		t.Fatalf("expected 1 received message, got %d", mockServer.MessageCount())
	}

	last := mockServer.LastMessage()
	if last == nil {
		t.Fatal("expected last message to be non-nil")
	}

	if last.Subject != "Test Plain Email" {
		t.Errorf("expected subject 'Test Plain Email', got '%s'", last.Subject)
	}
	if !strings.Contains(last.TextBody, "Hello, this is a plain text test email.") {
		t.Errorf("text body missing expected string: %s", last.TextBody)
	}
	if len(last.To) != 1 || last.To[0] != "recipient@example.com" {
		t.Errorf("recipient mismatch: %+v", last.To)
	}
}

func TestMockSMTPServer_HTMLAndAlternative(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "tickets@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &smtp.EmailMessage{
		To:       []string{"fan@tickid.com"},
		Subject:  "TickId Order Confirmation",
		TextBody: "Your order BK-1234 has been confirmed.",
		HTMLBody: "<h1>Your Order BK-1234</h1><p>Enjoy the concert!</p>",
	}

	err = client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("failed to send HTML email: %v", err)
	}

	last := mockServer.LastMessage()
	if last == nil {
		t.Fatal("expected message to be recorded")
	}

	if !strings.Contains(last.HTMLBody, "<h1>Your Order BK-1234</h1>") {
		t.Errorf("expected HTML content in body, got: %s", last.HTMLBody)
	}
	if !strings.Contains(last.TextBody, "Your order BK-1234 has been confirmed.") {
		t.Errorf("expected Text content in body, got: %s", last.TextBody)
	}
}

func TestMockSMTPServer_Attachment(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "tickets@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	fakePDF := []byte("%PDF-1.4\n1 0 obj\n<< /Title (E-Ticket) >>\nendobj\n%%EOF")

	msg := &smtp.EmailMessage{
		To:       []string{"attendee@example.com"},
		Subject:  "Your E-Ticket PDF",
		TextBody: "Please find your attached E-Ticket.",
		HTMLBody: "<p>Please find your attached E-Ticket.</p>",
		Attachments: []smtp.Attachment{
			{
				Filename:    "ticket.pdf",
				ContentType: "application/pdf",
				Data:        fakePDF,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("failed to send email with attachment: %v", err)
	}

	last := mockServer.LastMessage()
	if last == nil {
		t.Fatal("expected message recorded")
	}

	if len(last.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(last.Attachments))
	}

	att := last.Attachments[0]
	if att.Filename != "ticket.pdf" {
		t.Errorf("expected filename 'ticket.pdf', got '%s'", att.Filename)
	}
	if !bytes.Equal(att.Data, fakePDF) {
		t.Errorf("attachment content mismatch: expected %d bytes, got %d bytes", len(fakePDF), len(att.Data))
	}
}

func TestMockSMTPServer_WithAuthentication(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockServer.Close()

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "testuser", "testpassword", "auth@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &smtp.EmailMessage{
		To:       []string{"user@test.com"},
		Subject:  "Authenticated Email",
		TextBody: "Testing auth support.",
	}

	err = client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("failed to send authenticated email: %v", err)
	}

	found := mockServer.FindMessageBySubject("Authenticated")
	if found == nil {
		t.Fatal("expected message with subject 'Authenticated' to be found")
	}
}

func TestMockSMTPServer_HelperMethods(t *testing.T) {
	mockServer, err := smtp.NewMockSMTPServer()
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer mockServer.Close()

	if mockServer.Addr() == "" {
		t.Error("expected non-empty Addr()")
	}

	client, err := smtp.NewClient(mockServer.Host(), mockServer.Port(), "", "", "dispatch@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// Send to Alice
	_ = client.Send(ctx, &smtp.EmailMessage{
		To:       []string{"alice@tickid.com"},
		Subject:  "Hello Alice",
		TextBody: "Welcome Alice",
	})

	// Send to Bob
	_ = client.Send(ctx, &smtp.EmailMessage{
		To:       []string{"bob@tickid.com"},
		Subject:  "Hello Bob",
		TextBody: "Welcome Bob",
	})

	aliceMsgs := mockServer.FindMessagesTo("alice@tickid.com")
	if len(aliceMsgs) != 1 {
		t.Errorf("expected 1 message to Alice, got %d", len(aliceMsgs))
	}

	bobMsgs := mockServer.FindMessagesTo("bob@tickid.com")
	if len(bobMsgs) != 1 {
		t.Errorf("expected 1 message to Bob, got %d", len(bobMsgs))
	}

	rawMsgs := mockServer.GetRawMessages()
	if len(rawMsgs) != 2 {
		t.Errorf("expected 2 raw messages, got %d", len(rawMsgs))
	}

	allMsgs := mockServer.GetReceivedMessages()
	if len(allMsgs) != 2 {
		t.Errorf("expected 2 parsed messages, got %d", len(allMsgs))
	}

	// Test Reset
	mockServer.Reset()
	if mockServer.MessageCount() != 0 {
		t.Errorf("expected 0 messages after Reset, got %d", mockServer.MessageCount())
	}
	if mockServer.LastMessage() != nil {
		t.Error("expected nil LastMessage() after Reset")
	}
}

func TestClient_ValidationErrors(t *testing.T) {
	client, err := smtp.NewClient("127.0.0.1", 1025, "", "", "test@tickid.com", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// Nil message
	if err := client.Send(ctx, nil); err == nil {
		t.Error("expected error for nil message")
	}

	// No recipients
	if err := client.Send(ctx, &smtp.EmailMessage{Subject: "Test", TextBody: "Body"}); err == nil {
		t.Error("expected error for empty recipients")
	}

	// Empty subject
	if err := client.Send(ctx, &smtp.EmailMessage{To: []string{"test@example.com"}, TextBody: "Body"}); err == nil {
		t.Error("expected error for empty subject")
	}

	// Empty body
	if err := client.Send(ctx, &smtp.EmailMessage{To: []string{"test@example.com"}, Subject: "Test"}); err == nil {
		t.Error("expected error for empty body")
	}
}
