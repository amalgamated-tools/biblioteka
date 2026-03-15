package handlers

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
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

func isLoopbackHost(host string) bool {
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

// isValidSMTPHostForStatus performs a minimal validation suitable for deciding
// whether SMTP appears configured. It intentionally rejects obviously
// misformatted values such as "host:port", bracketed IPv6, or values with
// ASCII whitespace/control characters, while accepting plain hostnames or IPs.
func isValidSMTPHostForStatus(host string) bool {
	if host == "" {
		return false
	}

	// Disallow ASCII whitespace/control characters and bracketed IPv6.
	for _, r := range host {
		if r <= 0x20 || r == 0x7f {
			return false
		}
	}
	if strings.ContainsAny(host, "[]") {
		return false
	}

	// Allow bare IPv6 literals with colons (e.g. "::1"), but continue to reject
	// values that look like "host:port" or otherwise aren't valid IPs.
	if strings.Contains(host, ":") {
		if ip := net.ParseIP(host); ip == nil {
			return false
		}
	}

	return true
}

// ConfigHandler holds dependencies for configuration endpoints.
type ConfigHandler struct {
	DB               *db.DB
	IsOIDCConfigured func() bool
	OnOIDCConfigSet  func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error
	// SendMailFunc overrides the default sendMail implementation (used in tests).
	SendMailFunc func(ctx context.Context, addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error
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

	smtpCfg := h.resolveSMTPConfig(r.Context())

	host := strings.TrimSpace(smtpCfg.Host)
	from := strings.TrimSpace(smtpCfg.From)

	smtpConfigured := false
	if isValidSMTPHostForStatus(host) {
		if parsed, err := mail.ParseAddress(from); err == nil && parsed != nil && parsed.Address != "" && parsed.Name == "" {
			smtpConfigured = true
		}
	}

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

type smtpConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	From        string
	TLS         string
	EnvOverride bool
}

// resolveSMTPConfig returns the effective SMTP configuration.
// When SMTP_HOST is set as an environment variable, all fields are sourced
// from the environment (with defaults for port/tls). Otherwise, all fields
// come from the database.
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

// validateSMTPHost checks that host is a valid hostname/IP without embedded
// control characters, port numbers, or brackets.
// IPv6 addresses must be supplied without brackets (e.g. "::1" not "[::1]");
// net.JoinHostPort will add brackets automatically.
func validateSMTPHost(host string) error {
	if host == "" {
		return fmt.Errorf("host is required")
	}
	// Reject all ASCII control characters, space, and DEL to avoid malformed
	// addresses and potential log/header injection.
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c <= 0x20 || c == 0x7f {
			return fmt.Errorf("host contains invalid characters")
		}
	}
	if strings.ContainsAny(host, "[]") {
		return fmt.Errorf("host must not contain brackets; provide the bare IPv6 address")
	}
	// A bare IPv6 address contains colons — allow it only if net.ParseIP recognises it.
	if strings.Contains(host, ":") {
		if net.ParseIP(host) == nil {
			return fmt.Errorf("host must not contain a port; specify the port separately")
		}
	}
	return nil
}

// smtpSendParams holds the normalized, validated parameters ready for sending.
type smtpSendParams struct {
	Addr string    // host:port
	From string    // validated from address
	TLS  string    // normalized TLS mode
	Auth smtp.Auth // nil for unauthenticated SMTP
}

// validateSMTPForSend validates and normalizes an smtpConfig into parameters
// ready for sending. This is shared between handleSetSMTPConfig (for validation)
// and HandleSMTPTest (for validation + sending).
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

	return smtpSendParams{
		Addr: net.JoinHostPort(cfg.Host, port),
		From: from,
		TLS:  tlsMode,
		Auth: smtpAuth,
	}, nil
}

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

	cfg := h.resolveSMTPConfig(r.Context())

	writeJSON(r.Context(), w, http.StatusOK, smtpConfigResponse{
		Host:        cfg.Host,
		Port:        cfg.Port,
		Username:    cfg.Username,
		PasswordSet: cfg.Password != "",
		From:        cfg.From,
		TLS:         cfg.TLS,
		EnvOverride: cfg.EnvOverride,
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
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)

	// If password is empty but username is set, try to preserve the existing DB value,
	// but only when the stored username matches the requested username.
	// We intentionally do NOT fall back to SMTP_PASSWORD env var to avoid copying
	// env-managed secrets into the database.
	// When username is empty (switching to unauthenticated SMTP), we clear the
	// password to avoid leaving stale credentials in the database.
	if username == "" {
		password = ""
	} else if password == "" {
		// Load existing SMTP username to ensure we only reuse the password when
		// the username is unchanged.
		existingUsername, err := h.DB.GetSetting(r.Context(), settingSMTPUsername)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(
				r.Context(),
				"failed to load existing SMTP username",
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to load SMTP configuration")
			return
		}

		existingPassword, err := h.DB.GetSetting(r.Context(), settingSMTPPassword)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// No existing password stored; leave password empty and let
				// validateSMTPForSend enforce that a password is required.
			} else {
				slog.ErrorContext(
					r.Context(),
					"failed to load existing SMTP password",
					slog.Any(otelkeys.Error, err),
				)
				writeError(r.Context(), w, http.StatusInternalServerError, "failed to load SMTP configuration")
				return
			}
		} else if existingUsername == username && existingPassword != "" {
			// Only reuse the password when the username is unchanged to avoid
			// creating mismatched (username, password) pairs.
			password = existingPassword
		}
	}

	params, err := validateSMTPForSend(smtpConfig{
		Host:     host,
		Port:     strings.TrimSpace(req.Port),
		Username: username,
		Password: password,
		From:     strings.TrimSpace(req.From),
		TLS:      strings.TrimSpace(req.TLS),
	})
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}

	slog.DebugContext(r.Context(), "saving SMTP config",
		slog.String(otelkeys.Address, host),
		slog.String(otelkeys.Email, params.From),
	)

	// Extract normalized port from the validated addr (host:port).
	_, port, _ := net.SplitHostPort(params.Addr)

	for k, v := range map[string]string{
		settingSMTPHost:     host,
		settingSMTPPort:     port,
		settingSMTPUsername: username,
		settingSMTPPassword: password,
		settingSMTPFrom:     params.From,
		settingSMTPTLS:      params.TLS,
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
		"from": params.From,
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

	// Load SMTP config using the shared resolver.
	cfg := h.resolveSMTPConfig(r.Context())

	if cfg.Host == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "SMTP is not configured")
		return
	}
	if cfg.EnvOverride && cfg.From == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "incomplete SMTP environment configuration: SMTP_HOST is set but SMTP_FROM is missing")
		return
	}

	params, err := validateSMTPForSend(cfg)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}

	to := user.Email
	subject := "Biblioteka SMTP Test"
	body := "This is a test email from Biblioteka to confirm your SMTP settings are working correctly."
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", params.From, to, subject, body)

	send := sendMail
	if h.SendMailFunc != nil {
		send = h.SendMailFunc
	}

	if err := send(r.Context(), params.Addr, params.Auth, params.From, to, []byte(msg), params.TLS); err != nil {
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
// The context is propagated to dial calls so that cancelled requests
// (e.g. client disconnect or server shutdown) abort the SMTP session promptly.
func newSMTPClientWithContext(ctx context.Context, conn net.Conn, host string) (*smtp.Client, func(), error) {
	sessionDeadline := time.Now().Add(smtpSessionTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(sessionDeadline) {
		sessionDeadline = ctxDeadline
	}
	if err := conn.SetDeadline(sessionDeadline); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to set connection deadline: %w", err)
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

	cleanup := func() {
		close(done)
	}

	return client, cleanup, nil
}

func sendMail(ctx context.Context, addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
	netDialer := &net.Dialer{Timeout: 10 * time.Second}

	switch tlsMode {
	case "tls":
		tlsDialer := &tls.Dialer{
			NetDialer: netDialer,
			Config:    tlsConfig,
		}
		conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}

		client, cleanup, err := newSMTPClientWithContext(ctx, conn, host)
		if err != nil {
			return err
		}
		defer cleanup()
		defer client.Close()

		return smtpSend(client, a, from, to, msg)

	case "starttls":
		conn, err := netDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		sessionDeadline := time.Now().Add(smtpSessionTimeout)
		if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(sessionDeadline) {
			sessionDeadline = ctxDeadline
		}
		if err := conn.SetDeadline(sessionDeadline); err != nil {
			conn.Close()
			return fmt.Errorf("failed to set connection deadline: %w", err)
		}
		done := make(chan struct{})
		go func(c net.Conn, done <-chan struct{}, ctx context.Context) {
			select {
			case <-ctx.Done():
				c.Close()
			case <-done:
			}
		}(conn, done, ctx)
		defer close(done)
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
		conn, err := netDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}

		client, cleanup, err := newSMTPClientWithContext(ctx, conn, host)
		if err != nil {
			return err
		}
		defer cleanup()
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
