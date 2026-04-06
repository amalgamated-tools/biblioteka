package handlers

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Logout godoc
//
//	@Summary		Log out
//	@Description	Clears the authentication cookie
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	object{message=string}
//	@Failure		405	{object}	errorResponse
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !sameOrigin(r) {
		writeError(r.Context(), w, http.StatusForbidden, "invalid logout request origin")
		return
	}

	clearAuthCookie(w, h.SecureCookies)
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "logged out"})
}

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
	// Compare scheme.
	reqScheme := requestScheme(r)
	if !strings.EqualFold(u.Scheme, reqScheme) {
		return false
	}

	// Compare host (without port).
	originHost, originPort := parseHostPort(u.Host, defaultPort(u.Scheme))
	reqHost, reqPort := parseHostPort(r.Host, defaultPort(reqScheme))

	if !strings.EqualFold(originHost, reqHost) {
		return false
	}

	return originPort == reqPort
}

func defaultPort(scheme string) string {
	switch strings.ToLower(scheme) {
	case "https":
		return "443"
	default:
		return "80"
	}
}

func normalizeHost(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}
	return host
}

func parseHostPort(hostport, defaultPort string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		// Likely no port present; fall back to the provided default port.
		return normalizeHost(hostport), defaultPort
	}
	if port == "" {
		port = defaultPort
	}
	return host, port
}
