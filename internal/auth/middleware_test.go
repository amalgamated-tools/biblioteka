package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_MissingAuthorizationHeader(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "authentication required")
}

func TestMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Basic sometoken")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	// Non-Bearer auth header is treated as "no valid token found" since
	// extractToken checks header then cookie fallback.
	assertJSONError(t, w.Body.Bytes(), "authentication required")
}

func TestMiddleware_InvalidToken(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer notavalidtoken")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestMiddleware_ValidToken(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	token, _ := jm.CreateToken(t.Context(), "user-abc")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "user-abc" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "user-abc")
	}
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	ctx := context.Background()
	id := UserIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty user ID, got %q", id)
	}
}

func TestContextWithUserID(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithUserID(ctx, "test-user-123")
	got := UserIDFromContext(ctx)
	if got != "test-user-123" {
		t.Errorf("UserIDFromContext() = %q, want %q", got, "test-user-123")
	}
}

// assertJSONError checks that a JSON body contains the given error message.
func assertJSONError(t *testing.T, body []byte, wantMsg string) {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if resp["error"] != wantMsg {
		t.Errorf("error message = %q, want %q", resp["error"], wantMsg)
	}
}

// --- extractToken unit tests ---

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name       string
		header     string // Authorization header value ("" = not set)
		cookie     string // cookie value ("" = no cookie)
		wantToken  string
		wantSource tokenSource
	}{
		{
			name:       "no header no cookie",
			wantToken:  "",
			wantSource: tokenSourceNone,
		},
		{
			name:       "Bearer header only",
			header:     "Bearer validtoken",
			wantToken:  "validtoken",
			wantSource: tokenSourceHeader,
		},
		{
			name:       "cookie only",
			cookie:     "cookietoken",
			wantToken:  "cookietoken",
			wantSource: tokenSourceCookie,
		},
		{
			name:       "both present, header takes precedence",
			header:     "Bearer headertoken",
			cookie:     "cookietoken",
			wantToken:  "headertoken",
			wantSource: tokenSourceHeader,
		},
		{
			name:       "non-Bearer header with valid cookie falls back to cookie",
			header:     "Basic sometoken",
			cookie:     "cookietoken",
			wantToken:  "cookietoken",
			wantSource: tokenSourceCookie,
		},
		{
			name:       "non-Bearer header without cookie",
			header:     "Basic sometoken",
			wantToken:  "",
			wantSource: tokenSourceNone,
		},
		{
			name:       "Bearer with empty token falls back to cookie",
			header:     "Bearer ",
			cookie:     "cookietoken",
			wantToken:  "cookietoken",
			wantSource: tokenSourceCookie,
		},
		{
			name:       "Bearer with whitespace-only token falls back to cookie",
			header:     "Bearer   ",
			cookie:     "cookietoken",
			wantToken:  "cookietoken",
			wantSource: tokenSourceCookie,
		},
		{
			name:       "empty cookie value ignored",
			cookie:     "",
			wantToken:  "",
			wantSource: tokenSourceNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}
			if tt.cookie != "" {
				r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: tt.cookie})
			}
			got, gotSource, _ := extractToken(r)
			if got != tt.wantToken {
				t.Errorf("extractToken() token = %q, want %q", got, tt.wantToken)
			}
			if gotSource != tt.wantSource {
				t.Errorf("extractToken() source = %v, want %v", gotSource, tt.wantSource)
			}
		})
	}
}

// --- Cookie-based auth tests for Middleware ---

func TestMiddleware_ValidTokenViaCookie(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	token, _ := jm.CreateToken(t.Context(), "cookie-user")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: token})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "cookie-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "cookie-user")
	}
}

func TestMiddleware_InvalidTokenViaCookie(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "badtoken"})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestMiddleware_HeaderTakesPrecedenceOverCookie(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm, nil)

	headerToken, _ := jm.CreateToken(t.Context(), "header-user")
	cookieToken, _ := jm.CreateToken(t.Context(), "cookie-user")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+headerToken)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: cookieToken})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if gotUserID != "header-user" {
		t.Errorf("UserIDFromContext = %q, want %q (header should take precedence)", gotUserID, "header-user")
	}
}

// --- AdminMiddleware tests ---

// mockAdminChecker implements AdminChecker for testing.
type mockAdminChecker struct {
	admins map[string]bool
	err    error
}

func (m *mockAdminChecker) IsAdmin(_ context.Context, userID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.admins[userID], nil
}

func TestAdminMiddleware_NoToken(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	checker := &mockAdminChecker{admins: map[string]bool{}}
	mw := AdminMiddleware(jm, checker, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "authentication required")
}

func TestAdminMiddleware_InvalidToken(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	checker := &mockAdminChecker{admins: map[string]bool{}}
	mw := AdminMiddleware(jm, checker, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer badtoken")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestAdminMiddleware_NonAdminUser(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, nil)

	token, _ := jm.CreateToken(t.Context(), "regular-user")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called for non-admin")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "admin access required")
}

func TestAdminMiddleware_AdminUser(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, nil)

	token, _ := jm.CreateToken(t.Context(), "admin-user")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "admin-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "admin-user")
	}
}

func TestAdminMiddleware_AdminViaCookie(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, nil)

	token, _ := jm.CreateToken(t.Context(), "admin-user")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: token})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "admin-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "admin-user")
	}
}

func TestAdminMiddleware_CheckerError(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	checker := &mockAdminChecker{err: errors.New("db down")}
	mw := AdminMiddleware(jm, checker, nil)

	token, _ := jm.CreateToken(t.Context(), "some-user")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called on checker error")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "failed to verify permissions")
}

// --- API Key auth tests ---

// mockAPIKeyValidator implements APIKeyValidator for testing.
type mockAPIKeyValidator struct {
	keys    map[string]struct{ userID, keyID string } // keyHash -> (userID, keyID)
	touched []string                                  // keyIDs passed to TouchAPIKeyLastUsed
	err     error
}

func (m *mockAPIKeyValidator) ValidateAPIKey(_ context.Context, keyHash string) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	entry, ok := m.keys[keyHash]
	if !ok {
		return "", "", errors.New("api key not found")
	}
	return entry.userID, entry.keyID, nil
}

func (m *mockAPIKeyValidator) TouchAPIKeyLastUsed(_ context.Context, id string) error {
	m.touched = append(m.touched, id)
	return nil
}

func TestMiddleware_ValidAPIKey(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "apikey-user", keyID: "key-1"},
		},
	}
	mw := Middleware(jm, validator)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "apikey-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "apikey-user")
	}

	if len(validator.touched) != 1 || validator.touched[0] != "key-1" {
		t.Errorf("TouchAPIKeyLastUsed called with %v, want [key-1]", validator.touched)
	}
}

func TestMiddleware_InvalidAPIKey(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{},
	}
	mw := Middleware(jm, validator)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer bib_invalidkey00000000000000000000")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestMiddleware_APIKeyViaCookieRejected(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "apikey-user", keyID: "key-1"},
		},
	}
	mw := Middleware(jm, validator)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: apiKey})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called for API key via cookie")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestAdminMiddleware_ValidAPIKey(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "admin-user", keyID: "key-2"},
		},
	}
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, validator)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "admin-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "admin-user")
	}
}
