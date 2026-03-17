package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/coreos/go-oidc/v3/oidc"
)

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
	if !requireAdmin(h.DB, w, r) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching OIDC config", slog.String(otelkeys.UserID, userID))

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
	if !requireAdmin(h.DB, w, r) {
		return
	}

	var req setOIDCConfigRequest
	if !decodeJSON(r, w, &req) {
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

	for k, v := range map[string]string{
		settingOIDCIssuerURL:    issuerURL,
		settingOIDCClientID:     clientID,
		settingOIDCClientSecret: clientSecret,
		settingOIDCRedirectURI:  redirectURI,
	} {
		if err := h.DB.SetSetting(r.Context(), k, v); err != nil {
			slog.ErrorContext(r.Context(), "failed to save OIDC setting",
				slog.String(otelkeys.Key, k),
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to save OIDC configuration")
			return
		}
	}

	if h.OnOIDCConfigSet != nil {
		if err := h.OnOIDCConfigSet(r.Context(), issuerURL, clientID, clientSecret, redirectURI); err != nil {
			slog.ErrorContext(r.Context(), "failed to apply OIDC configuration", slog.Any(otelkeys.Error, err))
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
