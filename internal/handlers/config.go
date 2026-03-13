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
	"github.com/coreos/go-oidc/v3/oidc"
)

// const settingTMDBAPIKey = "tmdb_api_key"

const (
	settingOIDCIssuerURL    = "oidc_issuer_url"
	settingOIDCClientID     = "oidc_client_id"
	settingOIDCClientSecret = "oidc_client_secret"
	settingOIDCRedirectURI  = "oidc_redirect_uri"
)

// ConfigHandler holds dependencies for configuration endpoints.
type ConfigHandler struct {
	DB               *db.DB
	IsTMDBConfigured func() bool
	IsOIDCConfigured func() bool
	OnTMDBKeySet     func(key string) error
	OnOIDCConfigSet  func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error
}

type configStatusResponse struct {
	TMDBConfigured bool `json:"tmdb_configured"`
	OIDCConfigured bool `json:"oidc_configured"`
	IsAdmin        bool `json:"is_admin"`
}

type setTMDBKeyRequest struct {
	APIKey string `json:"api_key"`
}

// HandleConfigStatus handles GET /api/config/status
func (h *ConfigHandler) HandleConfigStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	isAdmin, _ := h.DB.IsAdmin(userID)

	writeJSON(w, http.StatusOK, configStatusResponse{
		TMDBConfigured: h.IsTMDBConfigured(),
		OIDCConfigured: h.IsOIDCConfigured(),
		IsAdmin:        isAdmin,
	})
}

// HandleSetTMDBKey handles PUT /api/config/tmdb-api-key
// Only the admin is allowed to change this setting.
func (h *ConfigHandler) HandleSetTMDBKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(userID)
	if err != nil {
		slog.Error("failed to check admin status", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "only the admin user can change this setting")
		return
	}

	var req setTMDBKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	// // Validate the key by creating a client and calling the TMDB API
	// client, err := tmdb.NewTMDBClient(apiKey)
	// if err != nil {
	// 	writeError(w, http.StatusBadRequest, "invalid TMDB API key")
	// 	return
	// }

	// cwr := client.ClientWithResponses()
	// resp, err := cwr.AuthenticationValidateKeyWithResponse(context.Background())
	// if err != nil {
	// 	slog.Error("failed to validate TMDB API key", "error", err)
	// 	writeError(w, http.StatusBadGateway, "failed to validate API key with TMDB")
	// 	return
	// }
	// if resp.JSON200 == nil || !resp.JSON200.Success {
	// 	writeError(w, http.StatusBadRequest, "invalid TMDB API key")
	// 	return
	// }

	// if err := h.DB.SetSetting(settingTMDBAPIKey, apiKey); err != nil {
	// 	slog.Error("failed to save TMDB API key", "error", err)
	// 	writeError(w, http.StatusInternalServerError, "failed to save API key")
	// 	return
	// }

	// if h.OnTMDBKeySet != nil {
	// 	if err := h.OnTMDBKeySet(apiKey); err != nil {
	// 		slog.Error("failed to apply TMDB API key", "error", err)
	// 		writeError(w, http.StatusInternalServerError, "key saved but failed to apply")
	// 		return
	// 	}
	// }

	writeJSON(w, http.StatusOK, map[string]string{"message": "TMDB API key configured successfully"})
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

// HandleGetOIDCConfig handles GET /api/config/oidc
// Only the admin user can view the OIDC configuration.
func (h *ConfigHandler) HandleGetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(userID)
	if err != nil {
		slog.Error("failed to check admin status", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "only the admin user can view this setting")
		return
	}

	issuerURL, _ := h.DB.GetSetting(settingOIDCIssuerURL)
	clientID, _ := h.DB.GetSetting(settingOIDCClientID)
	secret, secretErr := h.DB.GetSetting(settingOIDCClientSecret)
	redirectURI, _ := h.DB.GetSetting(settingOIDCRedirectURI)

	writeJSON(w, http.StatusOK, oidcConfigResponse{
		IssuerURL:       issuerURL,
		ClientID:        clientID,
		ClientSecretSet: secretErr == nil && secret != "",
		RedirectURI:     redirectURI,
	})
}

// HandleSetOIDCConfig handles PUT /api/config/oidc
// Only the admin user can change the OIDC configuration.
func (h *ConfigHandler) HandleSetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(userID)
	if err != nil {
		slog.Error("failed to check admin status", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "only the admin user can change this setting")
		return
	}

	var req setOIDCConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	issuerURL := strings.TrimSpace(req.IssuerURL)
	clientID := strings.TrimSpace(req.ClientID)
	clientSecret := strings.TrimSpace(req.ClientSecret)
	redirectURI := strings.TrimSpace(req.RedirectURI)

	if issuerURL == "" {
		writeError(w, http.StatusBadRequest, "issuer_url is required")
		return
	}
	if clientID == "" {
		writeError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	if redirectURI == "" {
		writeError(w, http.StatusBadRequest, "redirect_uri is required")
		return
	}

	// If client secret is empty, try to preserve the existing one
	if clientSecret == "" {
		existing, err := h.DB.GetSetting(settingOIDCClientSecret)
		if err != nil || existing == "" {
			writeError(w, http.StatusBadRequest, "client_secret is required")
			return
		}
		clientSecret = existing
	}

	// Validate the OIDC provider by performing discovery
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if _, err := oidc.NewProvider(ctx, issuerURL); err != nil {
		slog.Error("OIDC provider discovery failed", "issuer_url", issuerURL, "error", err)
		writeError(w, http.StatusBadRequest, "failed to discover OIDC provider at the given issuer URL")
		return
	}

	// Save all settings
	for k, v := range map[string]string{
		settingOIDCIssuerURL:    issuerURL,
		settingOIDCClientID:     clientID,
		settingOIDCClientSecret: clientSecret,
		settingOIDCRedirectURI:  redirectURI,
	} {
		if err := h.DB.SetSetting(k, v); err != nil {
			slog.Error("failed to save OIDC setting", "key", k, "error", err)
			writeError(w, http.StatusInternalServerError, "failed to save OIDC configuration")
			return
		}
	}

	// Apply the new configuration
	if h.OnOIDCConfigSet != nil {
		if err := h.OnOIDCConfigSet(r.Context(), issuerURL, clientID, clientSecret, redirectURI); err != nil {
			slog.Error("failed to apply OIDC configuration", "error", err)
			writeError(w, http.StatusInternalServerError, "settings saved but failed to apply OIDC configuration")
			return
		}
	}

	msg := "OIDC configuration saved successfully"
	if os.Getenv("OIDC_ISSUER_URL") != "" {
		msg = "OIDC settings saved. Note: the OIDC_ISSUER_URL environment variable is set and will take precedence. Remove OIDC_ISSUER_URL from the environment to use these settings."
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": msg})
}

// HandleOIDCConfig dispatches GET and PUT requests for /api/config/oidc.
func (h *ConfigHandler) HandleOIDCConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.HandleGetOIDCConfig(w, r)
	case http.MethodPut:
		h.HandleSetOIDCConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
