package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	netsmtp "net/smtp"
	"time"
)

// SessionTimeout is the maximum duration for a single SMTP session.
const SessionTimeout = 30 * time.Second

// newClientWithContext creates an SMTP client over conn, setting a deadline
// derived from ctx or SessionTimeout (whichever comes first). It returns the
// client, a cleanup function that must be called to release the underlying
// connection, and any error. On success the caller is responsible for calling
// both cleanup() and client.Close().
func newClientWithContext(ctx context.Context, conn net.Conn, host string) (*netsmtp.Client, func(), error) {
	sessionDeadline := time.Now().Add(SessionTimeout)
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
			c.Close()
		}
	}(conn, done, ctx)

	client, err := netsmtp.NewClient(conn, host)
	if err != nil {
		close(done)
		// Don't also call conn.Close() here; the goroutine (via done or ctx.Done())
		// is responsible for closing conn and is guaranteed to do so exactly once.
		return nil, nil, fmt.Errorf("SMTP client creation failed: %w", err)
	}

	return client, func() { close(done) }, nil
}

// Send dials addr, negotiates TLS according to tlsMode, authenticates with a
// if non-nil, and delivers a single message from → to.
func Send(ctx context.Context, addr string, a netsmtp.Auth, from, to string, msg []byte, tlsMode string) error {
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
		client, cleanup, err := newClientWithContext(ctx, conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()
		defer cleanup()
		return send(client, a, from, to, msg)
	case "starttls":
		conn, err := netDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		client, cleanup, err := newClientWithContext(ctx, conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()
		defer cleanup()
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
		return send(client, a, from, to, msg)
	case "none":
		conn, err := netDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		client, cleanup, err := newClientWithContext(ctx, conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()
		defer cleanup()
		return send(client, a, from, to, msg)
	default:
		return fmt.Errorf("unsupported TLS mode %q", tlsMode)
	}
}

func send(c *netsmtp.Client, a netsmtp.Auth, from, to string, msg []byte) error {
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
