// Package smtp provides helpers for configuring and sending email via SMTP.
// It is intentionally free of HTTP dependencies so that SMTP logic can be
// tested and reused independently of the HTTP handler layer.
package smtp

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	netsmtp "net/smtp"
	"os"
	"strconv"
	"strings"
)

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
	From string
	TLS  string
	Auth netsmtp.Auth
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

// ValidateHost checks that host is a syntactically valid SMTP hostname.
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host is required")
	}
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c <= 0x20 || c == 0x7f {
			return fmt.Errorf("host contains invalid characters")
		}
	}
	if strings.ContainsAny(host, "[]") {
		return fmt.Errorf("host must not contain brackets; provide the bare IPv6 address")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return fmt.Errorf("host must not contain a port; specify the port separately")
	}
	return nil
}

// ValidateForSend validates cfg and returns SendParams ready for a Send call.
func ValidateForSend(cfg Config) (SendParams, error) {
	if cfg.Host == "" {
		return SendParams{}, fmt.Errorf("host is required")
	}
	if err := ValidateHost(cfg.Host); err != nil {
		return SendParams{}, err
	}

	from := strings.TrimSpace(cfg.From)
	if from == "" {
		return SendParams{}, fmt.Errorf("from address is required")
	}
	if strings.ContainsAny(from, "\r\n") {
		return SendParams{}, fmt.Errorf("from address contains invalid characters")
	}
	parsedFrom, err := mail.ParseAddress(from)
	if err != nil {
		return SendParams{}, fmt.Errorf("from address is not a valid email address")
	}
	if parsedFrom.Name != "" {
		return SendParams{}, fmt.Errorf("from address must be a plain email address without a display name")
	}
	from = parsedFrom.Address

	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "587"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return SendParams{}, fmt.Errorf("port must be a number between 1 and 65535")
	}
	port = strconv.Itoa(portNum)

	tlsMode := strings.TrimSpace(cfg.TLS)
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	if tlsMode != "none" && tlsMode != "starttls" && tlsMode != "tls" {
		return SendParams{}, fmt.Errorf("tls must be one of: none, starttls, tls")
	}
	if cfg.Username != "" && cfg.Password == "" {
		return SendParams{}, fmt.Errorf("password is required when username is set")
	}
	if tlsMode == "none" && cfg.Username != "" && !IsLoopbackHost(cfg.Host) {
		return SendParams{}, fmt.Errorf("authenticated SMTP without TLS is only allowed for localhost/loopback; use STARTTLS or TLS for remote servers")
	}

	var smtpAuth netsmtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		smtpAuth = netsmtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	return SendParams{Addr: net.JoinHostPort(cfg.Host, port), From: from, TLS: tlsMode, Auth: smtpAuth}, nil
}
