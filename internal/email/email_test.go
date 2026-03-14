package email

import (
	"bytes"
	"context"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestNoopSender_ReturnsError(t *testing.T) {
	s := &NoopSender{}
	err := s.SendWithAttachment(context.Background(), "to@example.com", "subject", "body", "file.epub", []byte("data"), "application/epub+zip")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildMIMEMessage_Structure(t *testing.T) {
	from := "Biblioteka <noreply@example.com>"
	to := "reader@kindle.com"
	subject := "Your book: The Gunslinger"
	body := "Enjoy reading!"
	fileName := "the-gunslinger.epub"
	fileData := []byte("fake epub content")
	mimeType := "application/epub+zip"

	raw, err := buildMIMEMessage(from, to, subject, body, fileName, fileData, mimeType)
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("mail.ReadMessage: %v", err)
	}

	if got := msg.Header.Get("From"); got != from {
		t.Errorf("From = %q, want %q", got, from)
	}
	if got := msg.Header.Get("To"); got != to {
		t.Errorf("To = %q, want %q", got, to)
	}
	if got := msg.Header.Get("Subject"); got != subject {
		t.Errorf("Subject = %q, want %q", got, subject)
	}

	ct := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Errorf("media type = %q, want multipart/mixed", mediaType)
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	var partTypes []string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		partTypes = append(partTypes, part.Header.Get("Content-Type"))
	}

	if len(partTypes) != 2 {
		t.Fatalf("got %d MIME parts, want 2", len(partTypes))
	}
	if !strings.HasPrefix(partTypes[0], "text/plain") {
		t.Errorf("part[0] Content-Type = %q, want text/plain", partTypes[0])
	}
	if !strings.HasPrefix(partTypes[1], "application/epub+zip") {
		t.Errorf("part[1] Content-Type = %q, want application/epub+zip", partTypes[1])
	}
}

func TestBuildMIMEMessage_DefaultMimeType(t *testing.T) {
	raw, err := buildMIMEMessage("from@example.com", "to@example.com", "test", "body", "file.bin", []byte("data"), "")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("mail.ReadMessage: %v", err)
	}

	ct := msg.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	// skip text part
	if _, err := mr.NextPart(); err != nil {
		t.Fatalf("skip text part: %v", err)
	}
	attachPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("read attach part: %v", err)
	}
	if !strings.HasPrefix(attachPart.Header.Get("Content-Type"), "application/octet-stream") {
		t.Errorf("attach Content-Type = %q, want application/octet-stream", attachPart.Header.Get("Content-Type"))
	}
}

func TestEscapeHeaderValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"normal.epub", "normal.epub"},
		{`file"name.epub`, `file\"name.epub`},
		{`a"b"c`, `a\"b\"c`},
	}
	for _, tt := range tests {
		got := escapeHeaderValue(tt.in)
		if got != tt.want {
			t.Errorf("escapeHeaderValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNewSenderFromEnv_NoHost(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	s := NewSenderFromEnv()
	if _, ok := s.(*NoopSender); !ok {
		t.Errorf("expected NoopSender when SMTP_HOST is empty, got %T", s)
	}
}

func TestNewSenderFromEnv_WithHost(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "user@example.com")
	t.Setenv("SMTP_PASSWORD", "secret")
	t.Setenv("SMTP_FROM", "")
	s := NewSenderFromEnv()
	sender, ok := s.(*SMTPSender)
	if !ok {
		t.Fatalf("expected SMTPSender, got %T", s)
	}
	if sender.cfg.Host != "smtp.example.com" {
		t.Errorf("Host = %q, want smtp.example.com", sender.cfg.Host)
	}
	// SMTP_FROM defaults to SMTP_USERNAME when not set
	if sender.cfg.From != "user@example.com" {
		t.Errorf("From = %q, want user@example.com", sender.cfg.From)
	}
}

func TestNewSenderFromEnv_DefaultPort(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "")
	s := NewSenderFromEnv()
	sender, ok := s.(*SMTPSender)
	if !ok {
		t.Fatalf("expected SMTPSender, got %T", s)
	}
	if sender.cfg.Port != "587" {
		t.Errorf("Port = %q, want 587", sender.cfg.Port)
	}
}
