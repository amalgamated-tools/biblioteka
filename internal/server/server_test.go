package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
	goauthhandler "github.com/patrick-veverka/goauth/handler"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestDB(t *testing.T) *db.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "newTestDB: open")
	t.Cleanup(func() { _ = sqlDB.Close() })

	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err, "newTestDB: pragmas")

	err = db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite)
	require.NoError(t, err, "newTestDB: migrations")

	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

// ---------------------------------------------------------------------------
// staticCacheMiddleware
// ---------------------------------------------------------------------------

func TestStaticCacheMiddleware_AssetsImmutable(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := staticCacheMiddleware(inner)

	paths := []string{
		"/assets/index-Bm2A5K9v.js",
		"/assets/style-abc123.css",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equalf(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"),
				"expected immutable cache for hashed asset %s", path)
		})
	}
}

func TestStaticCacheMiddleware_IndexNoCache(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := staticCacheMiddleware(inner)

	for _, path := range []string{"/", "/index.html"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equalf(t, "no-cache", rec.Header().Get("Cache-Control"),
				"expected no-cache for entry point %s", path)
		})
	}
}

func TestStaticCacheMiddleware_OtherFiles_NoOverride(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := staticCacheMiddleware(inner)

	paths := []string{"/favicon.ico", "/assets/"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equalf(t, "", rec.Header().Get("Cache-Control"),
				"expected no Cache-Control override for %s", path)
		})
	}
}

// ---------------------------------------------------------------------------
// handleHealth
// ---------------------------------------------------------------------------

func TestHandleHealth_GET(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body healthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "ok", body.Status)
	ct := rec.Header().Get("Content-Type")
	require.Equal(t, "application/json", ct)
}

func TestHandleHealth_HEAD(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodHead, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleHealth_POST(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	allow := rec.Header().Get("Allow")
	require.NotEmpty(t, allow)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "method not allowed", body["error"])
}

// ---------------------------------------------------------------------------
// handleVersion
// ---------------------------------------------------------------------------

func TestHandleVersion_GET(t *testing.T) {
	s := &Server{version: "1.2.3"}
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body versionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "1.2.3", body.Version)
	ct := rec.Header().Get("Content-Type")
	require.Equal(t, "application/json", ct)
}

func TestHandleVersion_HEAD(t *testing.T) {
	s := &Server{version: "dev"}
	req := httptest.NewRequest(http.MethodHead, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleVersion_POST(t *testing.T) {
	s := &Server{version: "dev"}
	req := httptest.NewRequest(http.MethodPost, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	allow := rec.Header().Get("Allow")
	require.NotEmpty(t, allow)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "method not allowed", body["error"])
}

func TestHandleVersion_EmptyVersion(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body versionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "", body.Version)
}

// ---------------------------------------------------------------------------
// handleOIDCEnabled
// ---------------------------------------------------------------------------

func TestHandleOIDCEnabled_NotConfigured(t *testing.T) {
	s := &Server{} // oidcHandler is nil
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body enabledResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.False(t, body.Enabled)
}

func TestHandleOIDCEnabled_Configured(t *testing.T) {
	s := &Server{
		oidcHandler: &handlers.OIDCHandler{},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body enabledResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.True(t, body.Enabled)
}

func TestHandleOIDCEnabled_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	allow := rec.Header().Get("Allow")
	require.NotEmpty(t, allow)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "method not allowed", body["error"])
}

// ---------------------------------------------------------------------------
// handleSignupEnabled
// ---------------------------------------------------------------------------

func TestHandleSignupEnabled_Enabled(t *testing.T) {
	s := &Server{
		authHandler: &handlers.AuthHandler{AuthHandler: goauthhandler.AuthHandler{DisableSignup: false}},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/signup/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleSignupEnabled(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body enabledResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.True(t, body.Enabled)
}

func TestHandleSignupEnabled_Disabled(t *testing.T) {
	s := &Server{
		authHandler: &handlers.AuthHandler{AuthHandler: goauthhandler.AuthHandler{DisableSignup: true}},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/signup/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleSignupEnabled(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body enabledResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.False(t, body.Enabled)
}

func TestHandleSignupEnabled_MethodNotAllowed(t *testing.T) {
	s := &Server{
		authHandler: &handlers.AuthHandler{},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleSignupEnabled(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	allow := rec.Header().Get("Allow")
	require.NotEmpty(t, allow)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "method not allowed", body["error"])
}

// ---------------------------------------------------------------------------
// swaggerSecurityHeaders
// ---------------------------------------------------------------------------

func TestSwaggerSecurityHeaders_SetsHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := swaggerSecurityHeaders(inner)
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	checks := map[string]string{
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	}
	for header, want := range checks {
		got := rec.Header().Get(header)
		require.Equal(t, want, got, "header %s", header)
	}
}

func TestSwaggerSecurityHeaders_CrossOrigin(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := swaggerSecurityHeaders(inner)
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	require.Equal(t, "Origin", vary)

	require.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}

// ---------------------------------------------------------------------------
// oidcRoute
// ---------------------------------------------------------------------------

func TestOIDCRoute_NotConfigured(t *testing.T) {
	s := &Server{} // oidcHandler nil
	handler := s.oidcRoute((*handlers.OIDCHandler).Login)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	require.Equal(t, "OIDC not configured", body["error"])

	ct := rec.Header().Get("Content-Type")

	require.Equal(t, "application/json", ct)
}

func TestOIDCRoute_Configured(t *testing.T) {
	called := false
	fakeFn := func(_ *handlers.OIDCHandler, w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	s := &Server{
		oidcHandler: &handlers.OIDCHandler{},
	}
	handler := s.oidcRoute(fakeFn)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}

// ---------------------------------------------------------------------------
// NewServer with options
// ---------------------------------------------------------------------------

func TestNewServer_DefaultPort(t *testing.T) {
	d := newTestDB(t)

	s, err := NewServer(t.Context(), WithDB(d))
	require.NoError(t, err, "NewServer")

	require.Equal(t, 8080, s.port)
	require.Equal(t, "0.0.0.0:8080", s.Address)
}

func TestNewServer_WithPort(t *testing.T) {
	d := newTestDB(t)

	s, err := NewServer(t.Context(), WithDB(d), WithPort(9090))
	require.NoError(t, err, "NewServer")

	require.Equal(t, 9090, s.port)
	require.Equal(t, "0.0.0.0:9090", s.Address)
}

func TestNewServer_WithDB(t *testing.T) {
	d := newTestDB(t)

	s, err := NewServer(t.Context(), WithDB(d))
	require.NoError(t, err, "NewServer")

	require.Equal(t, d, s.DB)
}

// ---------------------------------------------------------------------------
// Route registration smoke tests
// ---------------------------------------------------------------------------

// newTestServer creates a full Server via NewServer with a test DB and returns
// an httptest.Server backed by the server's mux.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	d := newTestDB(t)
	s, err := NewServer(t.Context(), WithDB(d))
	require.NoError(t, err, "NewServer")
	ts := httptest.NewServer(s.mux)
	t.Cleanup(ts.Close)
	return ts
}

// TestRoutes_PublicEndpoints verifies that public (unauthenticated) system
// endpoints are reachable and return 200 OK.
func TestRoutes_PublicEndpoints(t *testing.T) {
	ts := newTestServer(t)

	routes := []string{
		"/api/health",
		"/api/version",
		"/api/auth/signup/enabled",
		"/api/auth/oidc/enabled",
	}

	for _, path := range routes {
		t.Run(path, func(t *testing.T) {
			resp, err := ts.Client().Get(ts.URL + path)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 for %s", path)
		})
	}
}

// TestRoutes_ProtectedEndpoints verifies that auth-protected API routes
// return 401 Unauthorized when no credentials are supplied.
func TestRoutes_ProtectedEndpoints(t *testing.T) {
	ts := newTestServer(t)

	routes := []string{
		"/api/books",
		"/api/authors",
		"/api/series",
		"/api/libraries",
		"/api/book-files/someID",
		"/api/audit-logs",
		"/api/api-keys",
		"/api/admin/users",
		"/api/config/status",
		"/api/auth/me",
	}

	for _, path := range routes {
		t.Run(path, func(t *testing.T) {
			resp, err := ts.Client().Get(ts.URL + path)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 for %s", path)
		})
	}
}
