package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockKoboTokenChecker struct {
	tokens map[string]*KoboTokenResult
}

func (m *mockKoboTokenChecker) GetKoboTokenByToken(_ context.Context, token string) (*KoboTokenResult, error) {
	if t, ok := m.tokens[token]; ok {
		return t, nil
	}
	return nil, sql.ErrNoRows
}

func TestKoboTokenAuthMiddleware_ValidToken(t *testing.T) {
	checker := &mockKoboTokenChecker{
		tokens: map[string]*KoboTokenResult{
			"abc123": {UserID: "user-1", Token: "abc123"},
		},
	}

	var gotPath, gotUserID, gotToken string
	handler := KoboTokenAuthMiddleware(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUserID = UserIDFromContext(r.Context())
		gotToken = KoboTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/kobo/abc123/v1/library/sync", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotPath != "/v1/library/sync" {
		t.Errorf("path = %q, want /v1/library/sync", gotPath)
	}
	if gotUserID != "user-1" {
		t.Errorf("userID = %q, want user-1", gotUserID)
	}
	if gotToken != "abc123" {
		t.Errorf("token = %q, want abc123", gotToken)
	}
}

func TestKoboTokenAuthMiddleware_InvalidToken(t *testing.T) {
	checker := &mockKoboTokenChecker{tokens: map[string]*KoboTokenResult{}}

	handler := KoboTokenAuthMiddleware(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for invalid token")
	}))

	r := httptest.NewRequest(http.MethodGet, "/kobo/badtoken/v1/initialization", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
}

func TestKoboTokenAuthMiddleware_EmptyToken(t *testing.T) {
	checker := &mockKoboTokenChecker{tokens: map[string]*KoboTokenResult{}}

	handler := KoboTokenAuthMiddleware(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for empty token")
	}))

	r := httptest.NewRequest(http.MethodGet, "/kobo/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestKoboTokenAuthMiddleware_TokenWithNoSubPath(t *testing.T) {
	checker := &mockKoboTokenChecker{
		tokens: map[string]*KoboTokenResult{
			"abc123": {UserID: "user-1", Token: "abc123"},
		},
	}

	var gotPath string
	handler := KoboTokenAuthMiddleware(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/kobo/abc123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotPath != "/" {
		t.Errorf("path = %q, want /", gotPath)
	}
}
