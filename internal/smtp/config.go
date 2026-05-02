// Package smtp provides helpers for configuring and sending email via SMTP.
// It is intentionally free of HTTP dependencies so that SMTP logic can be
// tested and reused independently of the HTTP handler layer.
package smtp

import (
	"context"
	"net"
	"net/mail"
	netsmtp "net/smtp"
	"os"
	"strconv"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/ssrf"
)

// ValidationError is returned by ValidateForSend when the smtp configuration
// is invalid. Its message is safe to surface directly to the API caller.
type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string { return e.msg }

func validationErr(msg string) error { return &ValidationError{msg: msg} }

// Setting key constants used to store SMTP configuration in the application
// settings table.
const (
	SettingKeyHost     = "smtp_host"
	SettingKeyPort     = "smtp_port"
	SettingKeyUsername = "smtp_username"
	SettingKeyPassword = "smtp_password"
	SettingKeyFrom     = "smtp_from"
	SettingKeyTLS      = "smtp_tls"
)

// Config holds SMTP connection settings.
type Config struct {
	Host        string
	Port        string
	Username    string
	Password    string
	From        string
	TLS         string
	EnvOverride bool
}

// SendParams holds validated parameters ready for sending an email.
type SendParams struct {
	Addr string
	// From is the bare email address used in the SMTP MAIL FROM envelope command.
	From string
	// FromHeader is the formatted address for use in the RFC 5322 "From:" message
	// header. It equals From when no display name is configured, or takes the form
	// "\"Display Name\" <addr>" when a display name is present.
	FromHeader string
	TLS        string
	Auth       netsmtp.Auth
}

// SendFunc is a function with the same signature as Send.
// ConfigHandler.SendMailFunc uses this type so tests can inject a fake sender.
type SendFunc func(ctx context.Context, addr string, auth netsmtp.Auth, from, to string, msg []byte, tlsMode string) error

// ResolveConfig reads SMTP settings, first checking environment variables,
// then falling back to the provided settings getter (typically db.DB.GetSetting).
func ResolveConfig(ctx context.Context, getSetting func(context.Context, string) (string, error)) Config {
	if os.Getenv("SMTP_HOST") != "" {
		port := os.Getenv("SMTP_PORT")
		if port == "" {
			port = "587"
		}
		tlsMode := os.Getenv("SMTP_TLS")
		if tlsMode == "" {
			tlsMode = "starttls"
		}
		return Config{
			Host:        os.Getenv("SMTP_HOST"),
			Port:        port,
			Username:    os.Getenv("SMTP_USERNAME"),
			Password:    os.Getenv("SMTP_PASSWORD"),
			From:        os.Getenv("SMTP_FROM"),
			TLS:         tlsMode,
			EnvOverride: true,
		}
	}

	host, _ := getSetting(ctx, SettingKeyHost)
	port, _ := getSetting(ctx, SettingKeyPort)
	username, _ := getSetting(ctx, SettingKeyUsername)
	password, _ := getSetting(ctx, SettingKeyPassword)
	from, _ := getSetting(ctx, SettingKeyFrom)
	tlsMode, _ := getSetting(ctx, SettingKeyTLS)

	return Config{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		TLS:      tlsMode,
	}
}

// IsLoopbackHost reports whether host is a loopback address (localhost or a
// loopback IP).
func IsLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// ValidateHost checks that host is a syntactically valid SMTP hostname and
// does not resolve to a private, loopback, or link-local address (SSRF protection).
func ValidateHost(host string) error {
	if host == "" {
		return validationErr("host is required")
	}
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c <= 0x20 || c == 0x7f {
			return validationErr("host contains invalid characters")
		}
	}
	if strings.ContainsAny(host, "[]") {
		return validationErr("host must not contain brackets; provide the bare IPv6 address")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return validationErr("host must not contain a port; specify the port separately")
	}
	// Block literal private/loopback/link-local IP addresses.
	if ip := net.ParseIP(host); ip != nil {
		if ssrf.IsPrivateIP(ip) {
			return validationErr("SMTP host must not be a private, loopback, or link-local address")
		}
		return nil
	}
	// Block well-known loopback hostnames (e.g. "localhost").
	if IsLoopbackHost(host) {
		return validationErr("SMTP host must not be a private, loopback, or link-local address")
	}
	return nil
}

// ValidateForSend validates cfg and returns SendParams ready for a Send call.
// All validation errors are returned as *ValidationError, which the caller may
// safely surface to API clients.
func ValidateForSend(cfg Config) (SendParams, error) {
	if cfg.Host == "" {
		return SendParams{}, validationErr("host is required")
	}
	if err := ValidateHost(cfg.Host); err != nil {
		return SendParams{}, err
	}

	from := strings.TrimSpace(cfg.From)
	if from == "" {
		return SendParams{}, validationErr("from address is required")
	}
	if strings.ContainsAny(from, "\r\n") {
		return SendParams{}, validationErr("from address contains invalid characters")
	}
	parsedFrom, err := mail.ParseAddress(from)
	if err != nil {
		return SendParams{}, validationErr("from address is not a valid email address")
	}
	// Use the bare email address for the SMTP MAIL FROM envelope command.
	// The display name (if any) is preserved in FromHeader for the message header.
	envelopeFrom := parsedFrom.Address
	var fromHeader string
	if parsedFrom.Name == "" {
		fromHeader = parsedFrom.Address
	} else {
		fromHeader = parsedFrom.String()
	}

	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "587"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return SendParams{}, validationErr("port must be a number between 1 and 65535")
	}
	port = strconv.Itoa(portNum)

	tlsMode := strings.TrimSpace(cfg.TLS)
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	if tlsMode != "none" && tlsMode != "starttls" && tlsMode != "tls" {
		return SendParams{}, validationErr("tls must be one of: none, starttls, tls")
	}
	if cfg.Username != "" && cfg.Password == "" {
		return SendParams{}, validationErr("password is required when username is set")
	}
	if tlsMode == "none" && cfg.Username != "" && !IsLoopbackHost(cfg.Host) {
		return SendParams{}, validationErr("authenticated SMTP without TLS is only allowed for localhost/loopback; use STARTTLS or TLS for remote servers")
	}

	var smtpAuth netsmtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		smtpAuth = netsmtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	return SendParams{Addr: net.JoinHostPort(cfg.Host, port), From: envelopeFrom, FromHeader: fromHeader, TLS: tlsMode, Auth: smtpAuth}, nil
}
