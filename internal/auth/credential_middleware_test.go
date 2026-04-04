package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"alice", "alice"},
		{"ALICE", "alice"},
		{"  Alice  ", "alice"},
		{"  ALICE  ", "alice"},
		{"", ""},
		{"  ", ""},
		{"MixedCase", "mixedcase"},
	}
	for _, tt := range tests {
		if got := NormalizeUsername(tt.input); got != tt.want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLookupByUsername_Found(t *testing.T) {
	type result struct {
		userID string
		hash   string
	}
	getFn := func(_ context.Context, username string) (*result, error) {
		if username == "alice" {
			return &result{userID: "user-1", hash: "hash-1"}, nil
		}
		return nil, sql.ErrNoRows
	}
	extract := func(r *result) (string, string) {
		return r.userID, r.hash
	}

	lookup := lookupByUsername(getFn, extract)

	userID, hash, err := lookup(t.Context(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want %q", userID, "user-1")
	}
	if hash != "hash-1" {
		t.Errorf("hash = %q, want %q", hash, "hash-1")
	}
}

func TestLookupByUsername_NotFound(t *testing.T) {
	type result struct{}
	getFn := func(_ context.Context, _ string) (*result, error) {
		return nil, sql.ErrNoRows
	}

	lookup := lookupByUsername(getFn, func(*result) (string, string) { return "", "" })

	_, _, err := lookup(t.Context(), "unknown")
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
}

func TestJSONErrorWriter(t *testing.T) {
	writer := jsonErrorWriter(http.StatusUnauthorized, "test error")

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	writer(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if !strings.Contains(w.Body.String(), "test error") {
		t.Errorf("body = %q, want it to contain %q", w.Body.String(), "test error")
	}
}

func TestJSONErrorWriter_ServiceUnavailable(t *testing.T) {
	writer := jsonErrorWriter(http.StatusServiceUnavailable, "Service temporarily unavailable")

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	writer(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(w.Body.String(), "Service temporarily unavailable") {
		t.Errorf("body = %q, want it to contain the message", w.Body.String())
	}
}
