package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMiddleware_MissingAuthorizationHeader(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
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
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
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
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
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
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
	mw := Middleware(jm, nil)

	token, err := jm.CreateToken(t.Context(), "user-abc")
	require.NoError(t, err, "CreateToken()")

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
	ctx := t.Context()
	id := UserIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty user ID, got %q", id)
	}
}

func TestContextWithUserID(t *testing.T) {
	ctx := t.Context()
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
		require.NoError(t, err, "failed to unmarshal response body")
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
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
	mw := Middleware(jm, nil)

	token, err := jm.CreateToken(t.Context(), "cookie-user")
	require.NoError(t, err, "CreateToken()")

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
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
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
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager()")
	mw := Middleware(jm, nil)

	headerToken, err := jm.CreateToken(t.Context(), "header-user")
	require.NoError(t, err, "CreateToken()")
	cookieToken, err := jm.CreateToken(t.Context(), "cookie-user")
	require.NoError(t, err, "CreateToken()")

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
