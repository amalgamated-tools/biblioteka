package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["error"] != "something went wrong" {
		t.Errorf("error = %q, want %q", result["error"], "something went wrong")
	}
}

func TestValidatePassword(t *testing.T) {
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

func TestExtractPathID(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		wantID string
		wantOK bool
	}{
		{"valid ID", "/api/movies/123", "/api/movies/", "123", true},
		{"trailing slash", "/api/movies/123/", "/api/movies/", "123", true},
		{"empty ID", "/api/movies/", "/api/movies/", "", false},
		{"sub-path", "/api/movies/123/details", "/api/movies/", "", false},
		{"alphanumeric ID", "/api/movies/abc-123", "/api/movies/", "abc-123", true},
		{"UUID-like ID", "/api/movies/550e8400-e29b-41d4-a716-446655440000", "/api/movies/", "550e8400-e29b-41d4-a716-446655440000", true},
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
