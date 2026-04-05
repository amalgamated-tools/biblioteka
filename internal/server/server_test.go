package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/handlers"
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

	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		_ = sqlDB.Close()
		require.NoError(t, err, "newTestDB: pragmas")
	}

	if err := db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite); err != nil {
		_ = sqlDB.Close()
		require.NoError(t, err, "newTestDB: migrations")
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func newTestJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "newTestJWT")
	return jm
}

// ---------------------------------------------------------------------------
// handleHealth
// ---------------------------------------------------------------------------

func TestHandleHealth_GET(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}

	var body healthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body.Status != "ok" {
		require.Failf(t, "failed", "expected status ok, got %q", body.Status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		require.Failf(t, "failed", "expected Content-Type application/json, got %q", ct)
	}
}

func TestHandleHealth_HEAD(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodHead, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}
}

func TestHandleHealth_POST(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		require.Failf(t, "failed", "expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		require.Fail(t, "expected Allow header to be set")
	}

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body["error"] != "method not allowed" {
		require.Failf(t, "failed", "expected error 'method not allowed', got %q", body["error"])
	}
}

// ---------------------------------------------------------------------------
// handleVersion
// ---------------------------------------------------------------------------

func TestHandleVersion_GET(t *testing.T) {
	s := &Server{version: "1.2.3"}
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}

	var body versionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body.Version != "1.2.3" {
		require.Failf(t, "failed", "expected version 1.2.3, got %q", body.Version)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		require.Failf(t, "failed", "expected Content-Type application/json, got %q", ct)
	}
}

func TestHandleVersion_HEAD(t *testing.T) {
	s := &Server{version: "dev"}
	req := httptest.NewRequest(http.MethodHead, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}
}

func TestHandleVersion_POST(t *testing.T) {
	s := &Server{version: "dev"}
	req := httptest.NewRequest(http.MethodPost, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		require.Failf(t, "failed", "expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		require.Fail(t, "expected Allow header to be set")
	}

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body["error"] != "method not allowed" {
		require.Failf(t, "failed", "expected error 'method not allowed', got %q", body["error"])
	}
}

func TestHandleVersion_EmptyVersion(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()

	s.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}

	var body versionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body.Version != "" {
		require.Failf(t, "failed", "expected empty version, got %q", body.Version)
	}
}

// ---------------------------------------------------------------------------
// handleOIDCEnabled
// ---------------------------------------------------------------------------

func TestHandleOIDCEnabled_NotConfigured(t *testing.T) {
	s := &Server{} // oidcHandler is nil
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}

	var body oidcEnabledResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body.Enabled {
		require.Fail(t, "expected enabled=false when oidcHandler is nil")
	}
}

func TestHandleOIDCEnabled_Configured(t *testing.T) {
	s := &Server{
		oidcHandler: &handlers.OIDCHandler{},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}

	var body oidcEnabledResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if !body.Enabled {
		require.Fail(t, "expected enabled=true when oidcHandler is non-nil")
	}
}

func TestHandleOIDCEnabled_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		require.Failf(t, "failed", "expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		require.Fail(t, "expected Allow header to be set")
	}

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body["error"] != "method not allowed" {
		require.Failf(t, "failed", "expected error 'method not allowed', got %q", body["error"])
	}
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

	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}

	checks := map[string]string{
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	}
	for header, want := range checks {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("header %s: expected %q, got %q", header, want, got)
		}
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
	if vary != "Origin" {
		t.Errorf("expected Vary: Origin, got %q", vary)
	}

	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %q", acao)
	}
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

	if rec.Code != http.StatusNotFound {
		require.Failf(t, "failed", "expected status 404, got %d", rec.Code)
	}

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "decode body")
	if body["error"] != "OIDC not configured" {
		require.Failf(t, "failed", "expected error 'OIDC not configured', got %q", body["error"])
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		require.Failf(t, "failed", "expected Content-Type application/json, got %q", ct)
	}
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

	if !called {
		require.Fail(t, "expected the OIDC handler function to be called")
	}
	if rec.Code != http.StatusOK {
		require.Failf(t, "failed", "expected status 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// NewServer with options
// ---------------------------------------------------------------------------

func TestNewServer_DefaultPort(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm))
	require.NoError(t, err, "NewServer")

	if s.port != 8080 {
		require.Failf(t, "failed", "expected default port 8080, got %d", s.port)
	}
	if s.Address != "0.0.0.0:8080" {
		require.Failf(t, "failed", "expected address 0.0.0.0:8080, got %s", s.Address)
	}
}

func TestNewServer_WithPort(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm), WithPort(9090))
	require.NoError(t, err, "NewServer")

	if s.port != 9090 {
		require.Failf(t, "failed", "expected port 9090, got %d", s.port)
	}
	if s.Address != "0.0.0.0:9090" {
		require.Failf(t, "failed", "expected address 0.0.0.0:9090, got %s", s.Address)
	}
}

func TestNewServer_WithDB(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm))
	require.NoError(t, err, "NewServer")

	if s.DB != d {
		require.Fail(t, "expected DB to be the injected test database")
	}
}

func TestNewServer_WithJWTManager(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm))
	require.NoError(t, err, "NewServer")

	if s.JWT != jm {
		require.Fail(t, "expected JWT to be the injected test JWT manager")
	}
}
