package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
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
		got := NormalizeUsername(tt.input)
		require.Equal(t, tt.want, got)
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
	require.NoError(t, err)
	require.Equal(t, "user-1", userID)
	require.Equal(t, "hash-1", hash)
}

func TestLookupByUsername_NotFound(t *testing.T) {
	type result struct{}
	getFn := func(_ context.Context, _ string) (*result, error) {
		return nil, sql.ErrNoRows
	}

	lookup := lookupByUsername(getFn, func(*result) (string, string) { return "", "" })

	_, _, err := lookup(t.Context(), "unknown")
	require.Error(t, err, "expected error for unknown user, got nil")
}

func TestJSONErrorWriter(t *testing.T) {
	writer := jsonErrorWriter(http.StatusUnauthorized, "test error")

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	writer(w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, "application/json", ct)
	require.Contains(t, w.Body.String(), "test error")
}

func TestJSONErrorWriter_ServiceUnavailable(t *testing.T) {
	writer := jsonErrorWriter(http.StatusServiceUnavailable, "Service temporarily unavailable")

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	writer(w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "Service temporarily unavailable")
}
