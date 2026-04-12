package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultPort(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		want   string
	}{
		{"https", "https", "443"},
		{"HTTPS", "HTTPS", "443"},
		{"Https", "Https", "443"},
		{"http", "http", "80"},
		{"HTTP", "HTTP", "80"},
		{"ftp", "ftp", "80"},
		{"(empty)", "", "80"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, defaultPort(tt.scheme))
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"127.0.0.1", "127.0.0.1"},
		{"[::1]", "::1"},
		{"[2001:db8::1]", "2001:db8::1"},
		// Missing closing bracket — no stripping
		{"[::1", "[::1"},
		// Missing opening bracket — no stripping
		{"::1]", "::1]"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeHost(tt.input))
		})
	}
}

func TestParseHostPort(t *testing.T) {
	tests := []struct {
		name        string
		hostport    string
		defaultPort string
		wantHost    string
		wantPort    string
	}{
		{
			name:     "hostname with port",
			hostport: "example.com:8080", defaultPort: "80",
			wantHost: "example.com", wantPort: "8080",
		},
		{
			name:     "hostname without port uses default",
			hostport: "example.com", defaultPort: "80",
			wantHost: "example.com", wantPort: "80",
		},
		{
			name:     "hostname without port uses https default",
			hostport: "example.com", defaultPort: "443",
			wantHost: "example.com", wantPort: "443",
		},
		{
			name:     "IPv4 with port",
			hostport: "127.0.0.1:3000", defaultPort: "80",
			wantHost: "127.0.0.1", wantPort: "3000",
		},
		{
			name:     "IPv4 without port",
			hostport: "127.0.0.1", defaultPort: "80",
			wantHost: "127.0.0.1", wantPort: "80",
		},
		{
			name:     "IPv6 with port",
			hostport: "[::1]:9000", defaultPort: "80",
			wantHost: "::1", wantPort: "9000",
		},
		{
			name:     "IPv6 without port",
			hostport: "[::1]", defaultPort: "80",
			wantHost: "::1", wantPort: "80",
		},
		{
			name:     "standard http port 80",
			hostport: "example.com:80", defaultPort: "80",
			wantHost: "example.com", wantPort: "80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := parseHostPort(tt.hostport, tt.defaultPort)
			require.Equal(t, tt.wantHost, host)
			require.Equal(t, tt.wantPort, port)
		})
	}
}

func TestSameOrigin(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		origin  string
		referer string
		proto   string
		want    bool
	}{
		{
			name:   "same origin via Origin header",
			host:   "example.com",
			origin: "http://example.com",
			want:   true,
		},
		{
			name:    "same origin via Referer fallback when no Origin",
			host:    "example.com",
			referer: "http://example.com/some/page",
			want:    true,
		},
		{
			name:   "cross-origin via Origin header",
			host:   "example.com",
			origin: "http://evil.com",
			want:   false,
		},
		{
			name:    "cross-origin via Referer fallback",
			host:    "example.com",
			referer: "http://evil.com/page",
			want:    false,
		},
		{
			name: "no Origin and no Referer",
			host: "example.com",
			want: false,
		},
		{
			name:   "invalid Origin URL rejected",
			host:   "example.com",
			origin: "://not-a-url",
			want:   false,
		},
		{
			name:    "invalid Referer URL rejected",
			host:    "example.com",
			referer: "://not-a-url",
			want:    false,
		},
		{
			name:   "Origin without host rejected",
			host:   "example.com",
			origin: "null",
			want:   false,
		},
		{
			name:   "different port",
			host:   "example.com:8080",
			origin: "http://example.com:9090",
			want:   false,
		},
		{
			name:   "explicit default port 80 matches bare host",
			host:   "example.com",
			origin: "http://example.com:80",
			want:   true,
		},
		{
			name:   "different subdomain",
			host:   "example.com",
			origin: "http://sub.example.com",
			want:   false,
		},
		{
			name:   "case-insensitive host comparison",
			host:   "Example.COM",
			origin: "http://example.com",
			want:   true,
		},
		{
			name:    "Origin header takes precedence over Referer",
			host:    "example.com",
			origin:  "http://evil.com",
			referer: "http://example.com/page",
			want:    false,
		},
		{
			name:   "same host with explicit port matches",
			host:   "example.com:8080",
			origin: "http://example.com:8080",
			want:   true,
		},
		{
			name:   "https origin rejected on http request",
			host:   "example.com",
			origin: "https://example.com",
			want:   false,
		},
		{
			name:   "https origin accepted with X-Forwarded-Proto https",
			host:   "example.com",
			origin: "https://example.com",
			proto:  "https",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				r.Header.Set("Referer", tt.referer)
			}
			if tt.proto != "" {
				r.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			require.Equal(t, tt.want, sameOrigin(r))
		})
	}
}

// TestLogout_CrossOrigin_Forbidden verifies that the logout endpoint rejects
// cross-origin POST requests, protecting against CSRF.
func TestLogout_CrossOrigin_Forbidden(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	r.Host = "example.com"
	r.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	h.Logout(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

// TestLogout_NoOriginOrReferer_Forbidden verifies that logout without Origin or
// Referer is also rejected (defensive-by-default).
func TestLogout_NoOriginOrReferer_Forbidden(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	// Deliberately no Origin or Referer headers.
	w := httptest.NewRecorder()
	h.Logout(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

// TestLogout_RefererFallback_OK verifies that when Origin is absent but a same-site
// Referer is present, logout still succeeds.
func TestLogout_RefererFallback_OK(t *testing.T) {
	h := newAuthHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	r.Host = "example.com"
	r.Header.Set("Referer", "http://"+r.Host+"/settings")
	w := httptest.NewRecorder()
	h.Logout(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}
