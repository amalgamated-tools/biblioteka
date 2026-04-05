package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
				if w.Code != http.StatusOK {
					t.Errorf("expected no response written, but got status %d", w.Code)
				}
				if w.Body.Len() != 0 {
					t.Errorf("expected empty body, but got %q", w.Body.String())
				}
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			var result map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "failed to unmarshal")
			if result["error"] != "name is required" {
				t.Errorf("error = %q, want %q", result["error"], "name is required")
			}
		})
	}
}

func Test_ValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantMsg  string
	}{
		{"valid password", "secret123", ""},
		{"exact minimum length", "123456", ""},
		{"too short", "abc", "password must be at least 6 characters"},
		{"empty", "", "password must be at least 6 characters"},
		{"one char", "x", "password must be at least 6 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePassword(tt.password)
			if got != tt.wantMsg {
				t.Errorf("validatePassword(%q) = %q, want %q", tt.password, got, tt.wantMsg)
			}
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
			if gotID != tt.wantID || gotSub != tt.wantSub || gotOK != tt.wantOK {
				t.Errorf("extractPathSegments(%q, %q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.path, tt.prefix, gotID, gotSub, gotOK, tt.wantID, tt.wantSub, tt.wantOK)
			}
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
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Errorf("extractPathID(%q, %q) = (%q, %v), want (%q, %v)",
					tt.path, tt.prefix, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}
