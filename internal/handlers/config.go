package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
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
)

// ConfigHandler holds dependencies for configuration endpoints.
type ConfigHandler struct {
	DB               *db.DB
	IsOIDCConfigured func() bool
	OnOIDCConfigSet  func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error
}

type configStatusResponse struct {
	OIDCConfigured bool `json:"oidc_configured"`
	IsAdmin        bool `json:"is_admin"`
}

// HandleConfigStatus godoc
// @Summary     Get configuration status
// @Description Returns OIDC configuration status and admin status
// @Tags        Config
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Success     200 {object} configStatusResponse
// @Router      /config/status [get]
func (h *ConfigHandler) HandleConfigStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching config status", slog.String(otelkeys.UserID, userID))
	isAdmin, _ := h.DB.IsAdmin(r.Context(), userID)

	writeJSON(r.Context(), w, http.StatusOK, configStatusResponse{
		OIDCConfigured: h.IsOIDCConfigured(),
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
// @Summary     Get OIDC configuration
// @Description Returns current OIDC configuration (admin only)
// @Tags        Config
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Success     200 {object} oidcConfigResponse
// @Failure     403 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /config/oidc [get]
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
// @Summary     Set OIDC configuration
// @Description Update OIDC configuration with validation (admin only)
// @Tags        Config
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       body body     setOIDCConfigRequest true "OIDC configuration"
// @Success     200  {object} object{message=string}
// @Failure     400  {object} errorResponse
// @Failure     403  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /config/oidc [put]
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
