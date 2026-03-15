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
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestDB(t *testing.T) *db.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("newTestDB: open: %v", err)
	}

	if _, err := sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("newTestDB: pragmas: %v", err)
	}

	if err := db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("newTestDB: migrations: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func newTestJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("testsecret", time.Hour)
	if err != nil {
		t.Fatalf("newTestJWT: %v", err)
	}
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
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %q", body.Status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}

func TestHandleHealth_HEAD(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodHead, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleHealth_POST(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		t.Fatal("expected Allow header to be set")
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "method not allowed" {
		t.Fatalf("expected error 'method not allowed', got %q", body["error"])
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
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body oidcEnabledResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Enabled {
		t.Fatal("expected enabled=false when oidcHandler is nil")
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
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body oidcEnabledResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body.Enabled {
		t.Fatal("expected enabled=true when oidcHandler is non-nil")
	}
}

func TestHandleOIDCEnabled_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/enabled", nil)
	rec := httptest.NewRecorder()

	s.handleOIDCEnabled(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		t.Fatal("expected Allow header to be set")
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "method not allowed" {
		t.Fatalf("expected error 'method not allowed', got %q", body["error"])
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
		t.Fatalf("expected status 200, got %d", rec.Code)
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
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "OIDC not configured" {
		t.Fatalf("expected error 'OIDC not configured', got %q", body["error"])
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
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
		t.Fatal("expected the OIDC handler function to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// NewServer with options
// ---------------------------------------------------------------------------

func TestNewServer_DefaultPort(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if s.port != 8080 {
		t.Fatalf("expected default port 8080, got %d", s.port)
	}
	if s.Address != "0.0.0.0:8080" {
		t.Fatalf("expected address 0.0.0.0:8080, got %s", s.Address)
	}
}

func TestNewServer_WithPort(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm), WithPort(9090))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if s.port != 9090 {
		t.Fatalf("expected port 9090, got %d", s.port)
	}
	if s.Address != "0.0.0.0:9090" {
		t.Fatalf("expected address 0.0.0.0:9090, got %s", s.Address)
	}
}

func TestNewServer_WithDB(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if s.DB != d {
		t.Fatal("expected DB to be the injected test database")
	}
}

func TestNewServer_WithJWTManager(t *testing.T) {
	d := newTestDB(t)
	jm := newTestJWT(t)

	s, err := NewServer(t.Context(), WithDB(d), WithJWTManager(jm))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if s.JWT != jm {
		t.Fatal("expected JWT to be the injected test JWT manager")
	}
}
