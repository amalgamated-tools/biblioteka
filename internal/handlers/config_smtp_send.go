package handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

const smtpSessionTimeout = 30 * time.Second

func newSMTPClientWithContext(ctx context.Context, conn net.Conn, host string) (*smtp.Client, func(), error) {
	sessionDeadline := time.Now().Add(smtpSessionTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(sessionDeadline) {
		sessionDeadline = ctxDeadline
	}
	if err := conn.SetDeadline(sessionDeadline); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	if err := ctx.Err(); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("context done before SMTP client creation: %w", err)
	}

	done := make(chan struct{})
	go func(c net.Conn, done <-chan struct{}, ctx context.Context) {
		select {
		case <-ctx.Done():
			c.Close()
		case <-done:
		}
	}(conn, done, ctx)

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		close(done)
		conn.Close()
		return nil, nil, fmt.Errorf("SMTP client creation failed: %w", err)
	}

	return client, func() { close(done) }, nil
}

func sendMail(ctx context.Context, addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	netDialer := &net.Dialer{Timeout: 10 * time.Second}

	switch tlsMode {
	case "tls":
		tlsDialer := &tls.Dialer{NetDialer: netDialer, Config: tlsConfig}
		conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		client, cleanup, err := newSMTPClientWithContext(ctx, conn, host)
		if err != nil {
			return err
		}
		defer client.Close()
		defer cleanup()
		return smtpSend(client, a, from, to, msg)
	case "starttls":
		conn, err := netDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		client, cleanup, err := newSMTPClientWithContext(ctx, conn, host)
		if err != nil {
			return err
		}
		defer client.Close()
		defer cleanup()
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
		return smtpSend(client, a, from, to, msg)
	case "none":
		conn, err := netDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		client, cleanup, err := newSMTPClientWithContext(ctx, conn, host)
		if err != nil {
			return err
		}
		defer client.Close()
		defer cleanup()
		return smtpSend(client, a, from, to, msg)
	default:
		return fmt.Errorf("unsupported TLS mode %q", tlsMode)
	}
}

func smtpSend(c *smtp.Client, a smtp.Auth, from, to string, msg []byte) error {
	if a != nil {
		if err := c.Auth(a); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("message write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("message close failed: %w", err)
	}
	return c.Quit()
}
