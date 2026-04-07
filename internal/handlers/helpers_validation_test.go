package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantCode  int
	}{
		{"valid name", "Stephen King", true, 0},
		{"empty string", "", false, http.StatusBadRequest},
		{"whitespace only", "   ", false, http.StatusBadRequest},
		{"tab only", "\t", false, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := validateName(t.Context(), w, tt.input)
			require.Equal(t, tt.wantValid, got, "validateName(%q)", tt.input)
			if tt.wantValid {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, 0, w.Body.Len())
				return
			}
			require.Equal(t, tt.wantCode, w.Code)
			var result map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
			require.Equal(t, "name is required", result["error"])
		})
	}
}

func Test_ValidatePassword(t *testing.T) {
	wantMinMsg := fmt.Sprintf("password must be at least %d characters", minPasswordLength)
	wantMaxMsg := fmt.Sprintf("password must be at most %d characters", maxPasswordLength)
	tests := []struct {
		name      string
		password  string
		wantValid bool
		wantMsg   string
	}{
		{"valid password", "secret123", true, ""},
		{"exact minimum length", "12345678", true, ""},
		{"too short", "abc", false, wantMinMsg},
		{"7 chars (one below minimum)", "1234567", false, wantMinMsg},
		{"empty", "", false, wantMinMsg},
		{"one char", "x", false, wantMinMsg},
		{"exact maximum length", strings.Repeat("a", maxPasswordLength), true, ""},
		{"one above maximum", strings.Repeat("a", maxPasswordLength+1), false, wantMaxMsg},
		{"well above maximum", strings.Repeat("a", 200), false, wantMaxMsg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := validatePassword(t.Context(), w, tt.password)
			require.Equal(t, tt.wantValid, got, "validatePassword(%q)", tt.password)
			if tt.wantValid {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, 0, w.Body.Len())
				return
			}
			require.Equal(t, http.StatusBadRequest, w.Code)
			var result map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
			require.Equal(t, tt.wantMsg, result["error"])
		})
	}
}

func Test_ExtractPathSegments(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		prefix  string
		wantID  string
		wantSub string
		wantOK  bool
	}{
		{"id only", "/api/books/123", "/api/books/", "123", "", true},
		{"id with trailing slash", "/api/books/123/", "/api/books/", "123", "", true},
		{"id with sub-resource", "/api/books/123/authors", "/api/books/", "123", "authors", true},
		{"id with sub-resource trailing slash", "/api/books/123/authors/", "/api/books/", "123", "authors", true},
		{"empty path after prefix", "/api/books/", "/api/books/", "", "", false},
		{"prefix only no slash", "/api/books", "/api/books/", "", "", false},
		{"uuid id with sub", "/api/books/550e8400-e29b-41d4-a716-446655440000/files", "/api/books/", "550e8400-e29b-41d4-a716-446655440000", "files", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotSub, gotOK := extractPathSegments(tt.path, tt.prefix)
			require.Equal(t, tt.wantID, gotID)
			require.Equal(t, tt.wantSub, gotSub)
			require.Equal(t, tt.wantOK, gotOK)
		})
	}
}

func Test_ExtractPathID(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		wantID string
		wantOK bool
	}{
		{"valid ID", "/api/items/123", "/api/items/", "123", true},
		{"trailing slash", "/api/items/123/", "/api/items/", "123", true},
		{"empty ID", "/api/items/", "/api/items/", "", false},
		{"sub-path", "/api/items/123/details", "/api/items/", "", false},
		{"alphanumeric ID", "/api/items/abc-123", "/api/items/", "abc-123", true},
		{"UUID-like ID", "/api/items/550e8400-e29b-41d4-a716-446655440000", "/api/items/", "550e8400-e29b-41d4-a716-446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := extractPathID(tt.path, tt.prefix)
			require.Equal(t, tt.wantID, gotID)
			require.Equal(t, tt.wantOK, gotOK)
		})
	}
}
