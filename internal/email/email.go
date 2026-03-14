// Package email provides a simple interface for sending emails with attachments.
package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
)

// Sender is the interface for sending emails with attachments.
type Sender interface {
	// SendWithAttachment sends an email to the given address with a single file
	// attachment. fileName is the display name, fileData is the raw bytes of the
	// file, and mimeType is the content type of the attachment.
	SendWithAttachment(ctx context.Context, to, subject, body, fileName string, fileData []byte, mimeType string) error
}

// SMTPConfig holds the settings required to connect to an SMTP relay.
type SMTPConfig struct {
	// Host is the SMTP server hostname (e.g. "smtp.gmail.com").
	Host string
	// Port is the SMTP server port (e.g. "587" for STARTTLS).
	Port string
	// Username is the SMTP authentication username.
	Username string
	// Password is the SMTP authentication password.
	Password string
	// From is the RFC 5321 "MAIL FROM" / "From:" header address
	// (e.g. "Biblioteka <noreply@example.com>").
	From string
}

// SMTPSender sends emails via SMTP with PLAIN authentication and STARTTLS.
type SMTPSender struct {
	cfg SMTPConfig
}

// NewSMTPSender creates an SMTPSender from the supplied config.
func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// NewSenderFromEnv returns a Sender configured from environment variables:
//
//	SMTP_HOST      – required; if empty a NoopSender is returned
//	SMTP_PORT      – defaults to "587"
//	SMTP_USERNAME  – SMTP auth username
//	SMTP_PASSWORD  – SMTP auth password
//	SMTP_FROM      – From address (defaults to SMTP_USERNAME)
func NewSenderFromEnv() Sender {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return &NoopSender{}
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = os.Getenv("SMTP_USERNAME")
	}
	return NewSMTPSender(SMTPConfig{
		Host:     host,
		Port:     port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     from,
	})
}

// SendWithAttachment sends an email with a single binary attachment.
func (s *SMTPSender) SendWithAttachment(_ context.Context, to, subject, body, fileName string, fileData []byte, mimeType string) error {
	addr := s.cfg.Host + ":" + s.cfg.Port

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	msg, err := buildMIMEMessage(s.cfg.From, to, subject, body, fileName, fileData, mimeType)
	if err != nil {
		return fmt.Errorf("build email message: %w", err)
	}

	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// buildMIMEMessage constructs a multipart/mixed MIME message with a text body
// and a single binary attachment.
func buildMIMEMessage(from, to, subject, body, fileName string, fileData []byte, mimeType string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	mw := multipart.NewWriter(&buf)
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + mw.Boundary() + "\"\r\n")
	buf.WriteString("\r\n")

	// Text body part
	th := make(textproto.MIMEHeader)
	th.Set("Content-Type", "text/plain; charset=utf-8")
	th.Set("Content-Transfer-Encoding", "quoted-printable")
	textPart, err := mw.CreatePart(th)
	if err != nil {
		return nil, err
	}
	qpw := quotedprintable.NewWriter(textPart)
	if _, err := qpw.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := qpw.Close(); err != nil {
		return nil, err
	}

	// Attachment part
	ah := make(textproto.MIMEHeader)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	ah.Set("Content-Type", mimeType+"; name=\""+escapeHeaderValue(fileName)+"\"")
	ah.Set("Content-Transfer-Encoding", "base64")
	ah.Set("Content-Disposition", "attachment; filename=\""+escapeHeaderValue(fileName)+"\"")
	attPart, err := mw.CreatePart(ah)
	if err != nil {
		return nil, err
	}
	enc := base64.NewEncoder(base64.StdEncoding, attPart)
	if _, err := enc.Write(fileData); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// escapeHeaderValue replaces double-quotes in a MIME header value to prevent
// header injection.
func escapeHeaderValue(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// NoopSender is returned when SMTP is not configured. Every call returns an
// error explaining that email is disabled.
type NoopSender struct{}

// SendWithAttachment always returns an error because SMTP is not configured.
func (*NoopSender) SendWithAttachment(_ context.Context, _, _, _, _ string, _ []byte, _ string) error {
	return fmt.Errorf("email sending is not configured: set SMTP_HOST to enable")
}
