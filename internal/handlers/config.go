package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/smtp"
)

const (
	settingOIDCIssuerURL    = "oidc_issuer_url"
	settingOIDCClientID     = "oidc_client_id"
	settingOIDCClientSecret = "oidc_client_secret"
	settingOIDCRedirectURI  = "oidc_redirect_uri"
)

func isValidSMTPHostForStatus(host string) bool {
	return smtp.ValidateHost(host) == nil
}

// ConfigHandler holds dependencies for configuration endpoints.
type ConfigHandler struct {
	DB               *db.DB
	IsOIDCConfigured func() bool
	OnOIDCConfigSet  func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error
	// IssuerURLValidator validates an OIDC issuer URL before provider discovery.
	// If nil, the default SSRF-aware validator (validateOIDCIssuerURL) is used.
	IssuerURLValidator func(ctx context.Context, rawURL string) error
	// OIDCHTTPClient overrides the HTTP client used for OIDC provider discovery.
	// If nil, an SSRF-safe client that blocks connections to private IPs is used.
	OIDCHTTPClient *http.Client
	// LLMEndpointURLValidator validates an LLM endpoint URL before it is stored.
	// If nil, the default SSRF-aware validator (validateLLMEndpointURL) is used.
	LLMEndpointURLValidator func(ctx context.Context, rawURL string) error
	// SendMailFunc overrides the default smtp.Send implementation (used in tests).
	SendMailFunc smtp.SendFunc
	// Secrets encrypts and decrypts sensitive settings (SMTP password, OIDC
	// client secret) stored in the database. If nil, values are stored as
	// plaintext (legacy behaviour preserved for backward compatibility).
	Secrets *auth.SecretEncrypter
}

type configStatusResponse struct {
	OIDCConfigured bool `json:"oidc_configured"`
	SMTPConfigured bool `json:"smtp_configured"`
	IsAdmin        bool `json:"is_admin"`
}

// HandleConfigStatus returns whether OIDC and SMTP are configured, and whether the authenticated user is an admin.
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

		slog.ErrorContext(
			r.Context(),
			"failed to check admin status",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "internal server error")
		return
	}

	smtpCfg := h.resolveSMTPConfig(r.Context())
	host := strings.TrimSpace(smtpCfg.Host)
	from := strings.TrimSpace(smtpCfg.From)

	smtpConfigured := false
	if isValidSMTPHostForStatus(host) {
		if parsed, err := mail.ParseAddress(from); err == nil && parsed != nil && parsed.Address != "" {
			smtpConfigured = true
		}
	}

	writeJSON(r.Context(), w, http.StatusOK, configStatusResponse{
		OIDCConfigured: h.IsOIDCConfigured(),
		SMTPConfigured: smtpConfigured,
		IsAdmin:        isAdmin,
	})
}

// resolveSMTPConfig reads the current SMTP configuration, preferring
// environment variables over database settings. If h.Secrets is set, the
// stored SMTP password is decrypted before use.
func (h *ConfigHandler) resolveSMTPConfig(ctx context.Context) smtp.Config {
	return smtp.ResolveConfig(ctx, makeDecryptingSMTPGetSetting(h.DB.GetSetting, h.Secrets))
}
