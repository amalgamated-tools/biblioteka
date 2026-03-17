package handlers

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

type smtpConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	From        string
	TLS         string
	EnvOverride bool
}

func (h *ConfigHandler) resolveSMTPConfig(ctx context.Context) smtpConfig {
	if os.Getenv("SMTP_HOST") != "" {
		port := os.Getenv("SMTP_PORT")
		if port == "" {
			port = "587"
		}
		tlsMode := os.Getenv("SMTP_TLS")
		if tlsMode == "" {
			tlsMode = "starttls"
		}
		return smtpConfig{
			Host:        os.Getenv("SMTP_HOST"),
			Port:        port,
			Username:    os.Getenv("SMTP_USERNAME"),
			Password:    os.Getenv("SMTP_PASSWORD"),
			From:        os.Getenv("SMTP_FROM"),
			TLS:         tlsMode,
			EnvOverride: true,
		}
	}

	host, _ := h.DB.GetSetting(ctx, settingSMTPHost)
	port, _ := h.DB.GetSetting(ctx, settingSMTPPort)
	username, _ := h.DB.GetSetting(ctx, settingSMTPUsername)
	password, _ := h.DB.GetSetting(ctx, settingSMTPPassword)
	from, _ := h.DB.GetSetting(ctx, settingSMTPFrom)
	tlsMode, _ := h.DB.GetSetting(ctx, settingSMTPTLS)

	return smtpConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		TLS:      tlsMode,
	}
}

func validateSMTPHost(host string) error {
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

type smtpSendParams struct {
	Addr string
	From string
	TLS  string
	Auth smtp.Auth
}

func validateSMTPForSend(cfg smtpConfig) (smtpSendParams, error) {
	if cfg.Host == "" {
		return smtpSendParams{}, fmt.Errorf("host is required")
	}
	if err := validateSMTPHost(cfg.Host); err != nil {
		return smtpSendParams{}, err
	}

	from := strings.TrimSpace(cfg.From)
	if from == "" {
		return smtpSendParams{}, fmt.Errorf("from address is required")
	}
	if strings.ContainsAny(from, "\r\n") {
		return smtpSendParams{}, fmt.Errorf("from address contains invalid characters")
	}
	parsedFrom, err := mail.ParseAddress(from)
	if err != nil {
		return smtpSendParams{}, fmt.Errorf("from address is not a valid email address")
	}
	if parsedFrom.Name != "" {
		return smtpSendParams{}, fmt.Errorf("from address must be a plain email address without a display name")
	}
	from = parsedFrom.Address

	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "587"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return smtpSendParams{}, fmt.Errorf("port must be a number between 1 and 65535")
	}
	port = strconv.Itoa(portNum)

	tlsMode := strings.TrimSpace(cfg.TLS)
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	if tlsMode != "none" && tlsMode != "starttls" && tlsMode != "tls" {
		return smtpSendParams{}, fmt.Errorf("tls must be one of: none, starttls, tls")
	}
	if cfg.Username != "" && cfg.Password == "" {
		return smtpSendParams{}, fmt.Errorf("password is required when username is set")
	}
	if tlsMode == "none" && cfg.Username != "" && !isLoopbackHost(cfg.Host) {
		return smtpSendParams{}, fmt.Errorf("authenticated SMTP without TLS is only allowed for localhost/loopback; use STARTTLS or TLS for remote servers")
	}

	var smtpAuth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		smtpAuth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	return smtpSendParams{Addr: net.JoinHostPort(cfg.Host, port), From: from, TLS: tlsMode, Auth: smtpAuth}, nil
}
