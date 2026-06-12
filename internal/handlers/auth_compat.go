// auth_compat.go provides handler-level auth helpers that bridge to goauth.
// This file replaces auth.go, auth_types.go, auth_cookies.go, auth_origin.go,
// and tokens.go which were deleted when switching to goauth.
package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/ssrf"
	goauthlib "github.com/amalgamated-tools/goauth/auth"
	goauthhandler "github.com/amalgamated-tools/goauth/handler"
	"github.com/coreos/go-oidc/v3/oidc"
)

// apiKeyDTO is the JSON representation of an API key (without the raw token).
// Defined here for swagger documentation; the actual serialization is performed
// by goauth's unexported types.
type apiKeyDTO struct { //nolint:unused
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// apiKeyCreateRequest is the request body for creating an API key.
// Defined here for swagger documentation; the actual decoding is performed by goauth.
type apiKeyCreateRequest struct { //nolint:unused
	Name string `json:"name"`
}

// apiKeyCreateResponse is the response body for a newly created API key.
// The raw key is returned only once and is never retrievable again.
// Defined here for swagger documentation; the actual serialization is performed by goauth.
type apiKeyCreateResponse struct { //nolint:unused
	apiKeyDTO
	Key string `json:"key"`
}

// AuthHandler wraps goauth's AuthHandler with biblioteka-specific methods (Logout)
// and audit logging for signup and password-change.
type AuthHandler struct {
	goauthhandler.AuthHandler
	DB *db.DB
}

// OIDCHandler wraps goauth's OIDCHandler with a registration gate on Callback.
type OIDCHandler struct {
	*goauthhandler.OIDCHandler
	DB *db.DB
}

// NewOIDCHandler creates a new OIDCHandler wrapping goauth's handler.
func NewOIDCHandler(ctx context.Context, users goauthlib.UserStore, jwt *goauthlib.JWTManager, issuerURL, clientID, clientSecret, redirectURI, cookieName string, secureCookies bool, database *db.DB) (*OIDCHandler, error) {
	safeCtx := oidc.ClientContext(ctx, ssrf.SafeHTTPClient(0))
	inner, err := goauthhandler.NewOIDCHandler(safeCtx, users, jwt, issuerURL, clientID, clientSecret, redirectURI, cookieName, secureCookies)
	if err != nil {
		return nil, err
	}
	return &OIDCHandler{OIDCHandler: inner, DB: database}, nil
}

// Callback wraps goauth's Callback, blocking user creation when registration is disabled.
// Note: this blocks ALL OIDC callbacks (including existing users) when registration_disabled is
// true, because the OIDC flow cannot distinguish new from returning users before user creation.
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.DB != nil {
		disabledStr, err := h.DB.GetSetting(r.Context(), db.SettingRegistrationDisabled)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(r.Context(), "failed to read registration_disabled setting; allowing OIDC callback", slog.Any(otelkeys.Error, err))
		}
		if disabled, _ := strconv.ParseBool(disabledStr); disabled {
			writeError(r.Context(), w, http.StatusForbidden, "signup is disabled")
			return
		}
	}
	h.OIDCHandler.Callback(w, r)
}

// PasskeyHandler wraps goauth's PasskeyHandler with audit logging for
// credential registration and deletion.
type PasskeyHandler struct {
	goauthhandler.PasskeyHandler
	DB *db.DB
}

// FinishRegistration wraps goauth's FinishRegistration to emit an audit log
// entry when a passkey credential is successfully created.
func (h *PasskeyHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	rc := newResponseCapture(w)
	h.PasskeyHandler.FinishRegistration(rc, r)
	if rc.status == http.StatusCreated && h.DB != nil {
		userID := auth.UserIDFromContext(r.Context())
		var resp struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rc.body.Bytes(), &resp); err != nil {
			slog.WarnContext(r.Context(), "failed to parse passkey registration response for audit",
				slog.Any(otelkeys.Error, err),
			)
		} else if resp.ID != "" {
			logAudit(r.Context(), h.DB, userID, db.AuditActionPasskeyCreated, "passkey", resp.ID, nil)
		}
	}
}

// DeleteCredential wraps goauth's DeleteCredential to emit an audit log entry
// when a passkey credential is successfully deleted.
func (h *PasskeyHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	id := h.URLParamFunc(r, "id")
	rc := newResponseCapture(w)
	h.PasskeyHandler.DeleteCredential(rc, r)
	if rc.status == http.StatusNoContent && h.DB != nil {
		userID := auth.UserIDFromContext(r.Context())
		logAudit(r.Context(), h.DB, userID, db.AuditActionPasskeyDeleted, "passkey", id, nil)
	}
}

// APIKeyHandler wraps goauth's APIKeyHandler with method-dispatching Handle
// methods so biblioteka's existing stdlib-mux routes continue to work, and
// with audit logging for create and delete.
type APIKeyHandler struct {
	goauthhandler.APIKeyHandler
	DB *db.DB
}

// Create wraps goauth's Create to emit an audit log entry on success.
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	rc := newResponseCapture(w)
	h.APIKeyHandler.Create(rc, r)
	if rc.status == http.StatusCreated && h.DB != nil {
		userID := auth.UserIDFromContext(r.Context())
		var resp struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(rc.body.Bytes(), &resp); err != nil {
			slog.WarnContext(r.Context(), "failed to parse API key create response for audit",
				slog.Any(otelkeys.Error, err),
			)
		} else if resp.ID != "" {
			logAudit(r.Context(), h.DB, userID, db.AuditActionAPIKeyCreated, "api_key", resp.ID, map[string]any{"name": resp.Name})
		} else {
			slog.WarnContext(r.Context(), "API key created but response ID is empty; audit skipped")
		}
	}
}

// Delete wraps goauth's Delete to emit an audit log entry on success.
func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := h.URLParamFunc(r, "id")
	rc := newResponseCapture(w)
	h.APIKeyHandler.Delete(rc, r)
	if rc.status == http.StatusNoContent && h.DB != nil {
		userID := auth.UserIDFromContext(r.Context())
		logAudit(r.Context(), h.DB, userID, db.AuditActionAPIKeyDeleted, "api_key", id, nil)
	}
}

// HandleAPIKeys dispatches GET (list) and POST (create) for /api/api-keys.
func (h *APIKeyHandler) HandleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleAPIKey dispatches DELETE for /api/api-keys/{id}.
//
//	@Summary		Delete an API key
//	@Description	Delete an API key by ID. Requests authenticated with this key will receive 401 afterward.
//	@Tags			api-keys
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	string	true	"API key ID"
//	@Success		204	"API key deleted"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"API key not found"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/api-keys/{id} [delete]
func (h *APIKeyHandler) HandleAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.Delete(w, r)
}

// clearAuthCookie clears the auth cookie.
func clearAuthCookie(w http.ResponseWriter, secure bool) {
	goauthhandler.ClearAuthCookie(w, auth.TokenCookieName(), secure)
}

// Signup wraps goauth's Signup to emit an audit log entry on success.
// If the registration_disabled setting is true, it returns 403 Forbidden
// before delegating to the underlying handler.
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if h.DB != nil {
		disabledStr, err := h.DB.GetSetting(r.Context(), db.SettingRegistrationDisabled)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(r.Context(), "failed to read registration_disabled setting; allowing signup", slog.Any(otelkeys.Error, err))
		}
		if disabled, _ := strconv.ParseBool(disabledStr); disabled {
			writeError(r.Context(), w, http.StatusForbidden, "signup is disabled")
			return
		}
	}
	rc := newResponseCapture(w)
	h.AuthHandler.Signup(rc, r)
	if rc.status == http.StatusCreated && h.DB != nil {
		var resp struct {
			User struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"user"`
		}
		if err := json.Unmarshal(rc.body.Bytes(), &resp); err != nil {
			slog.WarnContext(r.Context(), "failed to parse signup response for audit",
				slog.Any(otelkeys.Error, err),
			)
		} else if resp.User.ID != "" {
			logAudit(r.Context(), h.DB, resp.User.ID, db.AuditActionUserSignedUp, "user", resp.User.ID, map[string]any{
				"name":  resp.User.Name,
				"email": redactEmail(resp.User.Email),
			})
		}
	}
}

// ChangePassword wraps goauth's ChangePassword to emit an audit log entry on success.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	rc := newResponseCapture(w)
	h.AuthHandler.ChangePassword(rc, r)
	if rc.status == http.StatusOK && h.DB != nil {
		userID := auth.UserIDFromContext(r.Context())
		logAudit(r.Context(), h.DB, userID, db.AuditActionPasswordChanged, "user", userID, nil)
	}
}

// responseCapture wraps an http.ResponseWriter to capture the status code and
// response body while still writing through to the underlying writer.
type responseCapture struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func newResponseCapture(w http.ResponseWriter) *responseCapture {
	return &responseCapture{ResponseWriter: w}
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if rc.status == 0 {
		rc.status = http.StatusOK
	}
	rc.body.Write(b)
	return rc.ResponseWriter.Write(b)
}

// redactEmail partially obscures an email address for logging.
func redactEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

// generateRandomHex generates n random bytes as lowercase hex.
func generateRandomHex(n int) (string, error) {
	return auth.GenerateRandomHex(n)
}

// generateBase64Token generates a random Base64-encoded token.
func generateBase64Token(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("n must be positive")
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// sameOrigin checks if the request comes from the same origin.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		return matchRequestOrigin(u, r)
	}
	referer := r.Header.Get("Referer")
	if referer == "" {
		return false
	}
	u, err := url.Parse(referer)
	if err != nil || u.Host == "" {
		return false
	}
	return matchRequestOrigin(u, r)
}

func matchRequestOrigin(u *url.URL, r *http.Request) bool {
	reqScheme := requestScheme(r)
	if !strings.EqualFold(u.Scheme, reqScheme) {
		return false
	}
	originHost, originPort := parseHostPort(u.Host, defaultPort(u.Scheme))
	reqHost, reqPort := parseHostPort(r.Host, defaultPort(reqScheme))
	return strings.EqualFold(originHost, reqHost) && originPort == reqPort
}

func defaultPort(scheme string) string {
	if strings.ToLower(scheme) == "https" {
		return "443"
	}
	return "80"
}

func parseHostPort(hostport, defPort string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return normalizeHost(hostport), defPort
	}
	if port == "" {
		port = defPort
	}
	return host, port
}

// normalizeHost strips surrounding brackets from bracketed IPv6 literals like
// "[::1]" so host comparisons work uniformly.
func normalizeHost(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}
	return host
}
