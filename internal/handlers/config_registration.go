package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// registrationConfigResponse is the JSON shape for GET and PUT /api/config/registration.
type registrationConfigResponse struct {
	RegistrationDisabled bool `json:"registration_disabled"`
}

type setRegistrationConfigRequest struct {
	RegistrationDisabled bool `json:"registration_disabled"`
}

// HandleRegistrationConfig dispatches GET and PUT requests for /api/config/registration.
//
//	@Summary		Get or update registration configuration
//	@Description	GET returns whether public self-registration is disabled (admin only). PUT updates the setting (admin only).
//	@Tags			Config
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	registrationConfigResponse
//	@Failure		400	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		403	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/config/registration [get]
//	@Router			/config/registration [put]
func (h *ConfigHandler) HandleRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetRegistrationConfig(w, r)
	case http.MethodPut:
		h.handleSetRegistrationConfig(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ConfigHandler) handleGetRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching registration config", slog.String(otelkeys.UserID, userID))

	disabledStr := h.getSettingOrEmpty(r.Context(), db.SettingRegistrationDisabled)
	disabled, _ := strconv.ParseBool(disabledStr)

	writeJSON(r.Context(), w, http.StatusOK, registrationConfigResponse{
		RegistrationDisabled: disabled,
	})
}

func (h *ConfigHandler) handleSetRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	var req setRegistrationConfigRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if err := h.DB.SetSetting(r.Context(), db.SettingRegistrationDisabled, strconv.FormatBool(req.RegistrationDisabled)); err != nil {
		slog.ErrorContext(r.Context(), "failed to save registration config", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to save registration config")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionRegistrationConfigUpdated, "config", "registration", map[string]any{
		"registration_disabled": req.RegistrationDisabled,
	})

	writeJSON(r.Context(), w, http.StatusOK, registrationConfigResponse{
		RegistrationDisabled: req.RegistrationDisabled,
	})
}
