package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/coreos/go-oidc/v3/oidc"
)

// privateIPNets lists IP networks that must never be targeted by an OIDC
// provider discovery request (SSRF protection).
var privateIPNets = func() []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range []string{
		"10.0.0.0/8",     // RFC-1918 class A private
		"172.16.0.0/12",  // RFC-1918 class B private
		"192.168.0.0/16", // RFC-1918 class C private
		"127.0.0.0/8",    // IPv4 loopback
		"169.254.0.0/16", // IPv4 link-local (incl. AWS IMDS 169.254.169.254)
		"0.0.0.0/8",      // "this" network
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique-local (fc00:: / fd00::)
		"fe80::/10",      // IPv6 link-local
		"::/128",         // IPv6 unspecified
		"100.64.0.0/10",  // Shared address space (RFC 6598)
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("handlers: invalid CIDR %q: %v", cidr, err))
		}
		nets = append(nets, n)
	}
	return nets
}()

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateIPNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// dnsLookupTimeout is the maximum time allowed for a DNS lookup during
// issuer URL validation.
const dnsLookupTimeout = 2 * time.Second

// validateOIDCIssuerURL rejects issuer URLs that could be exploited for
// Server-Side Request Forgery (SSRF):
//   - only the https scheme is permitted
//   - userinfo (user:password) in the URL is rejected to prevent credential leakage
//   - literal private/loopback/link-local IP addresses in the host are blocked
//   - IPv6 literals with zone identifiers are rejected
//   - if the host is a DNS name, it is resolved (with a bounded timeout) and any
//     private/loopback/link-local address in the result is also blocked
func validateOIDCIssuerURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid issuer_url: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("issuer_url must use the https scheme")
	}
	if u.User != nil {
		return errors.New("issuer_url must not contain userinfo (credentials)")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("issuer_url must include a host")
	}

	// Reject IPv6 literals with zone identifiers (e.g. "fe80::1%lo0") which
	// can bypass net.ParseIP and fall through to DNS resolution.
	if strings.Contains(host, "%") {
		return errors.New("issuer_url must not contain an IPv6 zone identifier")
	}

	// Block literal private/loopback/link-local IP addresses directly in the URL.
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return errors.New("issuer_url must not point to a private, loopback, or link-local address")
		}
		return nil // a routable literal IP is accepted; no DNS lookup needed
	}

	// Resolve the hostname and block any private/loopback/link-local result.
	// Use a short timeout so a slow/hanging DNS server cannot block the
	// request indefinitely.
	//
	// DNS errors (timeout, NXDOMAIN, etc.) are intentionally swallowed here
	// (fail-open). This preserves availability: oidc.NewProvider will surface
	// the connectivity problem with its own request, and the SSRF-safe dialer
	// provides a second layer of defense at connect time.
	dnsCtx, cancel := context.WithTimeout(ctx, dnsLookupTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(dnsCtx, host)
	if err == nil {
		for _, addr := range addrs {
			if ip := net.ParseIP(addr); ip != nil && isPrivateIP(ip) {
				return errors.New("issuer_url must not resolve to a private, loopback, or link-local address")
			}
		}
	}
	return nil
}

// ssrfSafeHTTPClient returns an *http.Client whose dialer validates that the
// resolved IP address is not in a private/loopback/link-local range. This
// prevents DNS rebinding attacks where a hostname passes the initial
// validation but resolves to a private address at connect time.
func ssrfSafeHTTPClient() *http.Client {
	baseDialer := &net.Dialer{}
	safeDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		var safeIP string
		for _, ipStr := range ips {
			if ip := net.ParseIP(ipStr); ip != nil && isPrivateIP(ip) {
				return nil, fmt.Errorf("refusing to connect to private address %s", ipStr)
			} else if safeIP == "" {
				safeIP = ipStr
			}
		}
		if safeIP == "" {
			return nil, fmt.Errorf("no valid addresses for host %s", host)
		}
		// Connect directly to the validated IP — never re-resolve the hostname.
		return baseDialer.DialContext(ctx, network, net.JoinHostPort(safeIP, port))
	}
	return &http.Client{
		Transport: &http.Transport{DialContext: safeDialContext},
	}
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

// HandleGetOIDCConfig returns the current OIDC provider configuration (admin only).
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

// HandleSetOIDCConfig validates and persists a new OIDC provider configuration (admin only).
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

	// Validate the issuer URL to prevent SSRF attacks.
	validator := h.IssuerURLValidator
	if validator == nil {
		validator = validateOIDCIssuerURL
	}
	if err := validator(r.Context(), issuerURL); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}

	slog.DebugContext(r.Context(), "saving OIDC config",
		slog.String(otelkeys.IssuerURL, issuerURL),
		slog.String(otelkeys.RedirectURI, redirectURI),
	)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	// Use a safe HTTP client that validates resolved IPs at connect time to
	// prevent DNS rebinding attacks (TOCTOU between validation and discovery).
	httpClient := h.OIDCHTTPClient
	if httpClient == nil {
		httpClient = ssrfSafeHTTPClient()
	}
	safeCtx := oidc.ClientContext(ctx, httpClient)
	if _, err := oidc.NewProvider(safeCtx, issuerURL); err != nil {
		slog.ErrorContext(ctx, "OIDC provider discovery failed",
			slog.String(otelkeys.IssuerURL, issuerURL),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "failed to discover OIDC provider at the given issuer URL")
		return
	}

	if err := h.DB.SetSettings(r.Context(), []db.Setting{
		{Key: settingOIDCIssuerURL, Value: issuerURL},
		{Key: settingOIDCClientID, Value: clientID},
		{Key: settingOIDCClientSecret, Value: clientSecret},
		{Key: settingOIDCRedirectURI, Value: redirectURI},
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to save OIDC configuration", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to save OIDC configuration")
		return
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
