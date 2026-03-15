package handlers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	settingOIDCIssuerURL    = "oidc_issuer_url"
	settingOIDCClientID     = "oidc_client_id"
	settingOIDCClientSecret = "oidc_client_secret"
	settingOIDCRedirectURI  = "oidc_redirect_uri"

	settingSMTPHost     = "smtp_host"
	settingSMTPPort     = "smtp_port"
	settingSMTPUsername = "smtp_username"
	settingSMTPPassword = "smtp_password"
	settingSMTPFrom     = "smtp_from"
	settingSMTPTLS      = "smtp_tls"
)

// ConfigHandler holds dependencies for configuration endpoints.
type ConfigHandler struct {
	DB               *db.DB
	IsOIDCConfigured func() bool
	OnOIDCConfigSet  func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error
	// SendMailFunc overrides the default sendMail implementation (used in tests).
	SendMailFunc func(addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error
}

type configStatusResponse struct {
	OIDCConfigured bool `json:"oidc_configured"`
	SMTPConfigured bool `json:"smtp_configured"`
	IsAdmin        bool `json:"is_admin"`
}

// HandleConfigStatus godoc
//
//	@Summary		Get configuration status
//	@Description	Returns OIDC and SMTP configuration status and admin status
//	@Tags			Config
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{object}	configStatusResponse
//	@Router			/config/status [get]
func (h *ConfigHandler) HandleConfigStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching config status", slog.String(otelkeys.UserID, userID))
	isAdmin, _ := h.DB.IsAdmin(r.Context(), userID)

	smtpHost, _ := h.DB.GetSetting(r.Context(), settingSMTPHost)
	smtpFrom, _ := h.DB.GetSetting(r.Context(), settingSMTPFrom)
	smtpConfigured := (smtpHost != "" && smtpFrom != "") || (os.Getenv("SMTP_HOST") != "" && os.Getenv("SMTP_FROM") != "")

	writeJSON(r.Context(), w, http.StatusOK, configStatusResponse{
		OIDCConfigured: h.IsOIDCConfigured(),
		SMTPConfigured: smtpConfigured,
		IsAdmin:        isAdmin,
	})
}

type oidcConfigResponse struct {
	IssuerURL       string `json:"issuer_url"`
	ClientID        string `json:"client_id"`
	ClientSecretSet bool   `json:"client_secret_set"`
	RedirectURI     string `json:"redirect_uri"`
}

type setOIDCConfigRequest struct {
	IssuerURL    string `json:"issuer_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

// HandleGetOIDCConfig godoc
//
//	@Summary		Get OIDC configuration
//	@Description	Returns current OIDC configuration (admin only)
//	@Tags			Config
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{object}	oidcConfigResponse
//	@Failure		403	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/config/oidc [get]
func (h *ConfigHandler) HandleGetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching OIDC config", slog.String(otelkeys.UserID, userID))
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "only the admin user can view this setting")
		return
	}

	issuerURL, _ := h.DB.GetSetting(r.Context(), settingOIDCIssuerURL)
	clientID, _ := h.DB.GetSetting(r.Context(), settingOIDCClientID)
	secret, secretErr := h.DB.GetSetting(r.Context(), settingOIDCClientSecret)
	redirectURI, _ := h.DB.GetSetting(r.Context(), settingOIDCRedirectURI)

	writeJSON(r.Context(), w, http.StatusOK, oidcConfigResponse{
		IssuerURL:       issuerURL,
		ClientID:        clientID,
		ClientSecretSet: secretErr == nil && secret != "",
		RedirectURI:     redirectURI,
	})
}

// HandleSetOIDCConfig godoc
//
//	@Summary		Set OIDC configuration
//	@Description	Update OIDC configuration with validation (admin only)
//	@Tags			Config
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		setOIDCConfigRequest	true	"OIDC configuration"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	errorResponse
//	@Failure		403		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/config/oidc [put]
func (h *ConfigHandler) HandleSetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "only the admin user can change this setting")
		return
	}

	var req setOIDCConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return
	}

	issuerURL := strings.TrimSpace(req.IssuerURL)
	clientID := strings.TrimSpace(req.ClientID)
	clientSecret := strings.TrimSpace(req.ClientSecret)
	redirectURI := strings.TrimSpace(req.RedirectURI)

	if issuerURL == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "issuer_url is required")
		return
	}
	if clientID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "client_id is required")
		return
	}
	if redirectURI == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "redirect_uri is required")
		return
	}

	// If client secret is empty, try to preserve the existing one
	if clientSecret == "" {
		existing, err := h.DB.GetSetting(r.Context(), settingOIDCClientSecret)
		if err != nil || existing == "" {
			writeError(r.Context(), w, http.StatusBadRequest, "client_secret is required")
			return
		}
		clientSecret = existing
	}

	slog.DebugContext(r.Context(), "saving OIDC config",
		slog.String(otelkeys.IssuerURL, issuerURL),
		slog.String(otelkeys.RedirectURI, redirectURI),
	)

	// Validate the OIDC provider by performing discovery
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if _, err := oidc.NewProvider(ctx, issuerURL); err != nil {
		slog.ErrorContext(ctx, "OIDC provider discovery failed",
			slog.String(otelkeys.IssuerURL, issuerURL),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "failed to discover OIDC provider at the given issuer URL")
		return
	}

	// Save all settings
	for k, v := range map[string]string{
		settingOIDCIssuerURL:    issuerURL,
		settingOIDCClientID:     clientID,
		settingOIDCClientSecret: clientSecret,
		settingOIDCRedirectURI:  redirectURI,
	} {
		if err := h.DB.SetSetting(r.Context(), k, v); err != nil {
			slog.ErrorContext(ctx, "failed to save OIDC setting",
				slog.String(otelkeys.Key, k),
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to save OIDC configuration")
			return
		}
	}

	// Apply the new configuration
	if h.OnOIDCConfigSet != nil {
		if err := h.OnOIDCConfigSet(r.Context(), issuerURL, clientID, clientSecret, redirectURI); err != nil {
			slog.ErrorContext(ctx, "failed to apply OIDC configuration",
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "settings saved but failed to apply OIDC configuration")
			return
		}
	}

	msg := "OIDC configuration saved successfully"
	if os.Getenv("OIDC_ISSUER_URL") != "" {
		msg = "OIDC settings saved. Note: the OIDC_ISSUER_URL environment variable is set and will take precedence. Remove OIDC_ISSUER_URL from the environment to use these settings."
	}
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": msg})
}

// HandleOIDCConfig dispatches GET and PUT requests for /api/config/oidc.
func (h *ConfigHandler) HandleOIDCConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.HandleGetOIDCConfig(w, r)
	case http.MethodPut:
		h.HandleSetOIDCConfig(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// SMTP configuration types

type smtpConfigResponse struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	Username    string `json:"username"`
	PasswordSet bool   `json:"password_set"`
	From        string `json:"from"`
	TLS         string `json:"tls"`
	EnvOverride bool   `json:"env_override"`
}

type setSMTPConfigRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLS      string `json:"tls"`
}

// HandleSMTPConfig dispatches GET and PUT requests for /api/config/smtp.
//
// HandleSMTPConfig godoc
// @Summary     Get or update SMTP configuration
// @Description GET returns current SMTP config (admin only). PUT updates SMTP config (admin only).
// @Tags        Config
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} smtpConfigResponse
// @Failure     400 {object} errorResponse
// @Failure     401 {object} errorResponse
// @Failure     403 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /config/smtp [get]
// @Router      /config/smtp [put]
func (h *ConfigHandler) HandleSMTPConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetSMTPConfig(w, r)
	case http.MethodPut:
		h.handleSetSMTPConfig(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ConfigHandler) handleGetSMTPConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "only the admin user can view this setting")
		return
	}

	host, _ := h.DB.GetSetting(r.Context(), settingSMTPHost)
	port, _ := h.DB.GetSetting(r.Context(), settingSMTPPort)
	username, _ := h.DB.GetSetting(r.Context(), settingSMTPUsername)
	password, passwordErr := h.DB.GetSetting(r.Context(), settingSMTPPassword)
	from, _ := h.DB.GetSetting(r.Context(), settingSMTPFrom)
	tlsMode, _ := h.DB.GetSetting(r.Context(), settingSMTPTLS)

	// Environment variables take precedence over DB settings
	envOverride := os.Getenv("SMTP_HOST") != ""
	if envOverride {
		host = os.Getenv("SMTP_HOST")
		if v := os.Getenv("SMTP_PORT"); v != "" {
			port = v
		}
		if v := os.Getenv("SMTP_USERNAME"); v != "" {
			username = v
		}
		if os.Getenv("SMTP_PASSWORD") != "" {
			password = "set"
			passwordErr = nil
		}
		if v := os.Getenv("SMTP_FROM"); v != "" {
			from = v
		}
		if v := os.Getenv("SMTP_TLS"); v != "" {
			tlsMode = v
		}
	}

	writeJSON(r.Context(), w, http.StatusOK, smtpConfigResponse{
		Host:        host,
		Port:        port,
		Username:    username,
		PasswordSet: passwordErr == nil && password != "",
		From:        from,
		TLS:         tlsMode,
		EnvOverride: envOverride,
	})
}

func (h *ConfigHandler) handleSetSMTPConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "only the admin user can change this setting")
		return
	}

	var req setSMTPConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return
	}

	host := strings.TrimSpace(req.Host)
	port := strings.TrimSpace(req.Port)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	from := strings.TrimSpace(req.From)
	tlsMode := strings.TrimSpace(req.TLS)

	if host == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "host is required")
		return
	}
	if from == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "from address is required")
		return
	}
	if strings.ContainsAny(from, "\r\n") {
		writeError(r.Context(), w, http.StatusBadRequest, "from address contains invalid characters")
		return
	}
	parsedFrom, err := mail.ParseAddress(from)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "from address is not a valid email address")
		return
	}
	from = parsedFrom.Address
	if port == "" {
		port = "587"
	}
	parsedPort, err := strconv.Atoi(port)
	if err != nil || parsedPort < 1 || parsedPort > 65535 {
		writeError(r.Context(), w, http.StatusBadRequest, "port must be a number between 1 and 65535")
		return
	}
	// normalize port representation
	port = strconv.Itoa(parsedPort)
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	if tlsMode != "none" && tlsMode != "starttls" && tlsMode != "tls" {
		writeError(r.Context(), w, http.StatusBadRequest, "tls must be one of: none, starttls, tls")
		return
	}

	// If password is empty, try to preserve the existing one (like OIDC client_secret).
	// Only require a password when a username is set (allow unauthenticated SMTP).
	if password == "" {
		existing, _ := h.DB.GetSetting(r.Context(), settingSMTPPassword)
		if existing != "" {
			password = existing
		} else if envPw := strings.TrimSpace(os.Getenv("SMTP_PASSWORD")); envPw != "" {
			password = envPw
		}
	}
	if username != "" && password == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "password is required when username is set")
		return
	}

	slog.DebugContext(r.Context(), "saving SMTP config",
		slog.String(otelkeys.Address, host),
		slog.String(otelkeys.Email, from),
	)

	for k, v := range map[string]string{
		settingSMTPHost:     host,
		settingSMTPPort:     port,
		settingSMTPUsername: username,
		settingSMTPPassword: password,
		settingSMTPFrom:     from,
		settingSMTPTLS:      tlsMode,
	} {
		if err := h.DB.SetSetting(r.Context(), k, v); err != nil {
			slog.ErrorContext(r.Context(), "failed to save SMTP setting",
				slog.String(otelkeys.Key, k),
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to save SMTP configuration")
			return
		}
	}

	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionSMTPConfigUpdated, "config", "smtp", map[string]any{
		"host": host,
		"from": from,
	}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	msg := "SMTP configuration saved successfully"
	if os.Getenv("SMTP_HOST") != "" {
		msg = "SMTP settings saved. Note: the SMTP_HOST environment variable is set and will take precedence. Remove SMTP_HOST from the environment to use these settings."
	}
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": msg})
}

// HandleSMTPTest sends a test email to the admin user's email address.
//
// HandleSMTPTest godoc
// @Summary     Send SMTP test email
// @Description Sends a test email to the authenticated admin user's email address (admin only)
// @Tags        Config
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} object{message=string}
// @Failure     400 {object} errorResponse
// @Failure     401 {object} errorResponse
// @Failure     403 {object} errorResponse
// @Failure     502 {object} errorResponse
// @Router      /config/smtp/test [post]
func (h *ConfigHandler) HandleSMTPTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "admin access required")
		return
	}

	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get user", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get user details")
		return
	}

	// Validate recipient email to prevent header injection and invalid addresses.
	userEmail := strings.TrimSpace(user.Email)
	if userEmail == "" {
		slog.ErrorContext(r.Context(), "user email is empty",
			slog.String(otelkeys.UserID, userID),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "user email is not configured")
		return
	}
	if strings.ContainsAny(userEmail, "\r\n") {
		slog.ErrorContext(r.Context(), "user email contains forbidden control characters",
			slog.String(otelkeys.UserID, userID),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "invalid user email")
		return
	}
	parsedUserEmail, err := mail.ParseAddress(userEmail)
	if err != nil {
		slog.ErrorContext(r.Context(), "user email is not a valid address",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "invalid user email")
		return
	}
	// Use the bare address for SMTP envelope commands.
	user.Email = parsedUserEmail.Address

	// Load SMTP config: env vars first, then DB
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	tlsMode := os.Getenv("SMTP_TLS")

	// When SMTP_HOST is set via env, only require SMTP_FROM.
	// Port defaults to 587 if missing; username/password are optional.
	if host != "" && from == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "incomplete SMTP environment configuration: SMTP_HOST is set but SMTP_FROM is missing")
		return
	}

	if host == "" {
		host, _ = h.DB.GetSetting(r.Context(), settingSMTPHost)
		port, _ = h.DB.GetSetting(r.Context(), settingSMTPPort)
		username, _ = h.DB.GetSetting(r.Context(), settingSMTPUsername)
		password, _ = h.DB.GetSetting(r.Context(), settingSMTPPassword)
		from, _ = h.DB.GetSetting(r.Context(), settingSMTPFrom)
		tlsMode, _ = h.DB.GetSetting(r.Context(), settingSMTPTLS)
	}

	if host == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "SMTP is not configured")
		return
	}

	// Validate SMTP "from" address to prevent header injection and invalid addresses.
	from = strings.TrimSpace(from)
	if from == "" {
		slog.ErrorContext(r.Context(), "SMTP from address is empty")
		writeError(r.Context(), w, http.StatusBadRequest, "SMTP from address is not configured")
		return
	}
	if strings.ContainsAny(from, "\r\n") {
		slog.ErrorContext(r.Context(), "SMTP from address contains forbidden control characters")
		writeError(r.Context(), w, http.StatusBadRequest, "invalid SMTP from address")
		return
	}
	parsedFromAddr, err := mail.ParseAddress(from)
	if err != nil {
		slog.ErrorContext(r.Context(), "SMTP from address is not a valid email address",
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "invalid SMTP from address")
		return
	}
	from = parsedFromAddr.Address
	if port == "" {
		port = "587"
	}
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	if tlsMode != "none" && tlsMode != "starttls" && tlsMode != "tls" {
		writeError(r.Context(), w, http.StatusBadRequest, "SMTP TLS mode is invalid, must be one of: none, starttls, tls")
		return
	}

	to := user.Email
	subject := "Biblioteka SMTP Test"
	body := "This is a test email from Biblioteka to confirm your SMTP settings are working correctly."
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, to, subject, body)

	addr := net.JoinHostPort(host, port)
	var smtpAuth smtp.Auth
	if username != "" && password != "" {
		smtpAuth = smtp.PlainAuth("", username, password, host)
	}

	send := sendMail
	if h.SendMailFunc != nil {
		send = h.SendMailFunc
	}

	if err := send(addr, smtpAuth, from, to, []byte(msg), tlsMode); err != nil {
		slog.ErrorContext(r.Context(), "failed to send test email",
			slog.String(otelkeys.Email, to),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadGateway, "failed to send test email")
		return
	}

	slog.InfoContext(r.Context(), "test email sent", slog.String(otelkeys.Email, to))
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Test email sent to %s", to),
	})
}

// smtpSessionTimeout is the overall deadline for the entire SMTP session
// (after the TCP connection is established).
const smtpSessionTimeout = 30 * time.Second

// sendMail sends an email using the specified TLS mode.
func sendMail(addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	tlsConfig := &tls.Config{ServerName: host}

	switch tlsMode {
	case "tls":
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		conn.SetDeadline(time.Now().Add(smtpSessionTimeout))
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()
		return smtpSend(client, a, from, to, msg)

	case "starttls":
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		conn.SetDeadline(time.Now().Add(smtpSessionTimeout))
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
		return smtpSend(client, a, from, to, msg)

	case "none":
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		conn.SetDeadline(time.Now().Add(smtpSessionTimeout))
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()
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
