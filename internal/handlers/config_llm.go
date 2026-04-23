package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/ssrf"
)

// validateLLMEndpointURL rejects LLM endpoint URLs that could be exploited for
// Server-Side Request Forgery (SSRF):
//   - only the http and https schemes are permitted
//   - userinfo (user:password) in the URL is rejected to prevent credential leakage
//   - literal private/loopback/link-local IP addresses in the host are blocked
//   - IPv6 literals with zone identifiers are rejected
//   - if the host is a DNS name, it is resolved (with a bounded timeout) and any
//     private/loopback/link-local address in the result is also blocked
func validateLLMEndpointURL(ctx context.Context, rawURL string) error {
	return ssrf.ValidateURL(ctx, rawURL, "endpoint", []string{"http", "https"})
}

// LLMConfig is the response/request body for the LLM configuration endpoint.
type LLMConfig struct {
	Provider        string `json:"provider"`
	Endpoint        string `json:"endpoint"`
	Model           string `json:"model"`
	Enabled         bool   `json:"enabled"`
	RestartRequired bool   `json:"restart_required,omitempty"`
}

// HandleLLMConfig handles GET and PUT /api/config/llm (admin-only).
//
//	@Summary		Get or update LLM configuration
//	@Description	GET returns current LLM config (admin only). PUT updates LLM config (admin only). Changes require a server restart.
//	@Tags			Config
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	LLMConfig
//	@Failure		400	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		403	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/config/llm [get]
//	@Router			/config/llm [put]
func (h *ConfigHandler) HandleLLMConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetLLMConfig(w, r)
	case http.MethodPut:
		h.handleSetLLMConfig(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ConfigHandler) handleGetLLMConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching LLM config", slog.String(otelkeys.UserID, userID))

	provider, _ := h.DB.GetSetting(r.Context(), db.SettingLLMProvider)
	endpoint, _ := h.DB.GetSetting(r.Context(), db.SettingLLMEndpoint)
	model, _ := h.DB.GetSetting(r.Context(), db.SettingLLMModel)
	enabledStr, _ := h.DB.GetSetting(r.Context(), db.SettingLLMEnabled)
	enabled, _ := strconv.ParseBool(enabledStr)

	writeJSON(r.Context(), w, http.StatusOK, LLMConfig{
		Provider: provider,
		Endpoint: endpoint,
		Model:    model,
		Enabled:  enabled,
	})
}

func (h *ConfigHandler) handleSetLLMConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	var req LLMConfig
	if !decodeJSON(r, w, &req) {
		return
	}

	if req.Enabled && strings.TrimSpace(req.Endpoint) == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "endpoint is required when LLM is enabled")
		return
	}

	if req.Enabled && strings.TrimSpace(req.Model) == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "model is required when LLM is enabled")
		return
	}

	if req.Enabled && req.Provider != "" && !llm.IsSupported(req.Provider) {
		writeError(r.Context(), w, http.StatusBadRequest,
			fmt.Sprintf("unsupported provider %q; supported: %s", req.Provider, strings.Join(llm.SupportedProviders, ", ")))
		return
	}

	req.Model = strings.TrimSpace(req.Model)
	req.Endpoint = strings.TrimSpace(req.Endpoint)

	// Validate the endpoint URL to prevent SSRF attacks when enabled.
	if req.Enabled && req.Endpoint != "" {
		validator := h.LLMEndpointURLValidator
		if validator == nil {
			validator = validateLLMEndpointURL
		}
		if err := validator(r.Context(), req.Endpoint); err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, err.Error())
			return
		}
	}

	settings := []db.Setting{
		{Key: db.SettingLLMProvider, Value: req.Provider},
		{Key: db.SettingLLMEndpoint, Value: req.Endpoint},
		{Key: db.SettingLLMModel, Value: req.Model},
		{Key: db.SettingLLMEnabled, Value: strconv.FormatBool(req.Enabled)},
	}

	if err := h.DB.SetSettings(r.Context(), settings); err != nil {
		slog.ErrorContext(r.Context(), "failed to save LLM config",
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to save LLM config")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionLLMConfigUpdated, "config", "llm", nil)

	writeJSON(r.Context(), w, http.StatusOK, LLMConfig{
		Provider:        req.Provider,
		Endpoint:        req.Endpoint,
		Model:           req.Model,
		Enabled:         req.Enabled,
		RestartRequired: true,
	})
}
