// auth_compat.go provides handler-level auth helpers that bridge to goauth.
// This file replaces auth.go, auth_types.go, auth_cookies.go, auth_origin.go,
// and tokens.go which were deleted when switching to goauth.
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	goauthhandler "github.com/amalgamated-tools/goauth/handler"
)

// AuthHandler wraps goauth's AuthHandler with biblioteka-specific methods (Logout).
type AuthHandler struct {
	goauthhandler.AuthHandler
}

// OIDCHandler wraps goauth's OIDCHandler.
type OIDCHandler = goauthhandler.OIDCHandler

// NewOIDCHandler delegates to goauth.
var NewOIDCHandler = goauthhandler.NewOIDCHandler

// PasskeyHandler wraps goauth's PasskeyHandler.
type PasskeyHandler = goauthhandler.PasskeyHandler

// APIKeyHandler wraps goauth's APIKeyHandler with method-dispatching Handle
// methods so biblioteka's existing stdlib-mux routes continue to work.
type APIKeyHandler struct {
	goauthhandler.APIKeyHandler
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
