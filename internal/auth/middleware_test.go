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

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
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

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
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

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "user-abc", gotUserID)
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	ctx := t.Context()
	id := UserIDFromContext(ctx)
	require.Equal(t, "", id)
}

func TestContextWithUserID(t *testing.T) {
	ctx := t.Context()
	ctx = ContextWithUserID(ctx, "test-user-123")
	got := UserIDFromContext(ctx)
	require.Equal(t, "test-user-123", got)
}

// assertJSONError checks that a JSON body contains the given error message.
func assertJSONError(t *testing.T, body []byte, wantMsg string) {
	t.Helper()
	var resp map[string]string
	err := json.Unmarshal(body, &resp)
	require.NoError(t, err, "failed to unmarshal response body")
	require.Equal(t, wantMsg, resp["error"])
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
			require.Equal(t, tt.wantToken, got)
			require.Equal(t, tt.wantSource, gotSource)
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

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "cookie-user", gotUserID)
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

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
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

	require.Equal(t, "header-user", gotUserID)
}
