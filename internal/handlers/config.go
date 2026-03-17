package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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

func isValidSMTPHostForStatus(host string) bool {
	if host == "" {
		return false
	}

	for _, r := range host {
		if r <= 0x20 || r == 0x7f {
			return false
		}
	}
	if strings.ContainsAny(host, "[]") {
		return false
	}

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

	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User not found (e.g., stale JWT) — signal client to re-authenticate.
			writeError(r.Context(), w, http.StatusUnauthorized, "unauthorized")
			return
		}

		slog.ErrorContext(r.Context(), "failed to check admin status", slog.String(otelkeys.UserID, userID))
		writeError(r.Context(), w, http.StatusInternalServerError, "internal server error")
		return
	}

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
