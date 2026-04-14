package smtp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// failDeadlineConn wraps a net.Conn and makes SetDeadline always fail.
type failDeadlineConn struct {
	net.Conn
}

func (f *failDeadlineConn) SetDeadline(_ time.Time) error {
	return fmt.Errorf("intentional SetDeadline failure")
}

// serveOnePeer writes the SMTP 220 greeting on conn and then drains reads
// until the connection is closed.
func serveOnePeer(conn net.Conn) {
	fmt.Fprintf(conn, "220 test.example.com ESMTP\r\n")
	buf := make([]byte, 256)
	for {
		if _, err := conn.Read(buf); err != nil {
			return
		}
	}
}

// newFakeSMTPServer starts a minimal in-process SMTP server on a random
// loopback port. It accepts a single connection, handles the full SMTP
// conversation, and sends the delivered message body into the returned channel.
// The listener is closed automatically at test cleanup.
func newFakeSMTPServer(t *testing.T) (addr string, delivered <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		runFakeSMTP(conn, ch)
	}()

	return ln.Addr().String(), ch
}

// runFakeSMTP drives a single SMTP conversation on conn and sends the
// delivered message body into ch.
func runFakeSMTP(conn net.Conn, ch chan<- string) {
	r := bufio.NewReader(conn)
	writeln := func(s string) { fmt.Fprintf(conn, "%s\r\n", s) }

	writeln("220 localhost ESMTP")

	var body strings.Builder
	inData := false

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				ch <- body.String()
				writeln("250 OK")
			} else {
				// Dot-unstuffing: a leading ".." becomes ".".
				if strings.HasPrefix(line, "..") {
					line = line[1:]
				}
				body.WriteString(line)
				body.WriteString("\n")
			}
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			writeln("250 localhost")
		case strings.HasPrefix(upper, "MAIL FROM"):
			writeln("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			writeln("250 OK")
		case upper == "DATA":
			inData = true
			writeln("354 End data with <CR><LF>.<CR><LF>")
		case upper == "QUIT":
			writeln("221 Bye")
			return
		default:
			writeln("500 Unrecognized command")
		}
	}
}

// ---- newClientWithContext tests ----

func TestNewClientWithContext_Success(t *testing.T) {
	server, client := net.Pipe()
	go serveOnePeer(server)

	ctx := context.Background()
	smtpClient, cleanup, err := newClientWithContext(ctx, client, "test.example.com")
	require.NoError(t, err)
	require.NotNil(t, smtpClient)
	require.NotNil(t, cleanup)
	defer smtpClient.Close()
	defer cleanup()
}

func TestNewClientWithContext_CancelledContext(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	// client will be closed inside newClientWithContext after the ctx.Err() check.

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call so ctx.Err() is non-nil

	_, cleanup, err := newClientWithContext(ctx, client, "test.example.com")
	require.Error(t, err)
	require.Nil(t, cleanup)
	require.ErrorContains(t, err, "context done before SMTP client creation")
}

func TestNewClientWithContext_SetDeadlineFails(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	// Wrap client so that SetDeadline always fails; newClientWithContext will
	// close the wrapped conn and return an error.
	wrapped := &failDeadlineConn{Conn: client}

	ctx := context.Background()
	_, cleanup, err := newClientWithContext(ctx, wrapped, "test.example.com")
	require.Error(t, err)
	require.Nil(t, cleanup)
	require.ErrorContains(t, err, "failed to set connection deadline")
}

func TestNewClientWithContext_ContextDeadlineEarlierThanSession(t *testing.T) {
	server, client := net.Pipe()
	go serveOnePeer(server)

	// Deadline of 10 s is shorter than SessionTimeout (30 s) but long enough
	// for the test to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	smtpClient, cleanup, err := newClientWithContext(ctx, client, "test.example.com")
	require.NoError(t, err)
	require.NotNil(t, smtpClient)
	defer smtpClient.Close()
	defer cleanup()
}

// ---- Send tests ----

func TestSend_InvalidAddress(t *testing.T) {
	err := Send(context.Background(), "notanaddr", nil, "from@example.com", "to@example.com", []byte("hello"), "none")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid address")
}

func TestSend_UnsupportedTLSMode(t *testing.T) {
	err := Send(context.Background(), "localhost:25", nil, "from@example.com", "to@example.com", []byte("hello"), "badmode")
	require.Error(t, err)
	require.ErrorContains(t, err, "unsupported TLS mode")
}

func TestSend_None_DeliverySuccess(t *testing.T) {
	addr, delivered := newFakeSMTPServer(t)

	msg := []byte("Subject: Unit test\r\n\r\nHello, world.")
	err := Send(context.Background(), addr, nil, "from@example.com", "to@example.com", msg, "none")
	require.NoError(t, err)

	select {
	case body := <-delivered:
		require.Contains(t, body, "Hello, world.")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for delivered message")
	}
}
