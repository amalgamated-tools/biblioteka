package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

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
	if !requireAdmin(h.DB, w, r) {
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
	if !requireAdmin(h.DB, w, r) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	var req setSMTPConfigRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	host := strings.TrimSpace(req.Host)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" {
		password = ""
	} else if password == "" {
		existingUsername, err := h.DB.GetSetting(r.Context(), settingSMTPUsername)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(r.Context(), "failed to load existing SMTP username", slog.Any(otelkeys.Error, err))
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to load SMTP configuration")
			return
		}

		existingPassword, err := h.DB.GetSetting(r.Context(), settingSMTPPassword)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				slog.ErrorContext(r.Context(), "failed to load existing SMTP password", slog.Any(otelkeys.Error, err))
				writeError(r.Context(), w, http.StatusInternalServerError, "failed to load SMTP configuration")
				return
			}
		} else if existingUsername == username && existingPassword != "" {
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

	_, port, _ := net.SplitHostPort(params.Addr)
	if err := h.DB.SetSettings(r.Context(), []db.Setting{
		{Key: settingSMTPHost, Value: host},
		{Key: settingSMTPPort, Value: port},
		{Key: settingSMTPUsername, Value: username},
		{Key: settingSMTPPassword, Value: password},
		{Key: settingSMTPFrom, Value: params.From},
		{Key: settingSMTPTLS, Value: params.TLS},
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to save SMTP configuration", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to save SMTP configuration")
		return
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

	if !requireAdmin(h.DB, w, r) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get user", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get user details")
		return
	}

	userEmail := strings.TrimSpace(user.Email)
	if userEmail == "" {
		slog.ErrorContext(r.Context(), "user email is empty", slog.String(otelkeys.UserID, userID))
		writeError(r.Context(), w, http.StatusBadRequest, "user email is not configured")
		return
	}
	if strings.ContainsAny(userEmail, "\r\n") {
		slog.ErrorContext(r.Context(), "user email contains forbidden control characters", slog.String(otelkeys.UserID, userID))
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
	user.Email = parsedUserEmail.Address

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
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		params.From,
		to,
		"Biblioteka SMTP Test",
		"This is a test email from Biblioteka to confirm your SMTP settings are working correctly.",
	)

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
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Test email sent to %s", to)})
}
