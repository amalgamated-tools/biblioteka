package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func Test_WriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(t.Context(), w, http.StatusOK, map[string]string{"key": "value"})

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

func Test_WriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(t.Context(), w, http.StatusBadRequest, "something went wrong")

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

func Test_HandleNameErr(t *testing.T) {
	var (
		errInvalid = errors.New("invalid name")
		errExists  = errors.New("name exists")
		errOther   = errors.New("other error")
	)

	tests := []struct {
		name        string
		err         error
		resourceArt string
		wantHandled bool
		wantCode    int
		wantErrMsg  string
	}{
		{
			name:        "nil error not handled",
			err:         nil,
			resourceArt: "an author",
			wantHandled: false,
		},
		{
			name:        "errInvalid yields 400",
			err:         errInvalid,
			resourceArt: "an author",
			wantHandled: true,
			wantCode:    http.StatusBadRequest,
			wantErrMsg:  "name is required",
		},
		{
			name:        "wrapped errInvalid yields 400",
			err:         fmt.Errorf("db: %w", errInvalid),
			resourceArt: "a series",
			wantHandled: true,
			wantCode:    http.StatusBadRequest,
			wantErrMsg:  "name is required",
		},
		{
			name:        "wrapped errExists yields 409",
			err:         fmt.Errorf("db: %w", errExists),
			resourceArt: "an author",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "an author with that name already exists",
		},
		{
			name:        "errExists yields 409",
			err:         errExists,
			resourceArt: "an author",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "an author with that name already exists",
		},
		{
			name:        "errExists series yields 409",
			err:         errExists,
			resourceArt: "a series",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "a series with that name already exists",
		},
		{
			name:        "unrelated error not handled",
			err:         errOther,
			resourceArt: "an author",
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := handleNameErr(t.Context(), w, tt.err, errInvalid, errExists, tt.resourceArt)
			if got != tt.wantHandled {
				t.Fatalf("handleNameErr() = %v, want %v", got, tt.wantHandled)
			}
			if !tt.wantHandled {
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
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if result["error"] != tt.wantErrMsg {
				t.Errorf("error = %q, want %q", result["error"], tt.wantErrMsg)
			}
		})
	}
}

func Test_HandleDBErr(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		resource    string
		wantHandled bool
		wantCode    int
		wantMsg     string
	}{
		{
			name:        "nil error",
			err:         nil,
			resource:    "book",
			wantHandled: false,
		},
		{
			name:        "not found",
			err:         sql.ErrNoRows,
			resource:    "author",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantMsg:     "author not found",
		},
		{
			name:        "wrapped not found",
			err:         fmt.Errorf("lookup failed: %w", sql.ErrNoRows),
			resource:    "series",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantMsg:     "series not found",
		},
		{
			name:        "other error",
			err:         errors.New("connection refused"),
			resource:    "library",
			wantHandled: true,
			wantCode:    http.StatusInternalServerError,
			wantMsg:     "failed to get library",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := handleDBErr(t.Context(), w, tt.err, tt.resource)
			if got != tt.wantHandled {
				t.Fatalf("handleDBErr() = %v, want %v", got, tt.wantHandled)
			}
			if !tt.wantHandled {
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			var result map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if result["error"] != tt.wantMsg {
				t.Errorf("error = %q, want %q", result["error"], tt.wantMsg)
			}
		})
	}
}

func Test_LogAudit(t *testing.T) {
	d := newTestDB(t)

	logAudit(t.Context(), d, "user-42", db.AuditActionBookCreated, "book", "book-1", map[string]any{"title": "Audited"})

	logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}
	if logs[0].UserID == nil || *logs[0].UserID != "user-42" {
		t.Errorf("user id = %v, want %q", logs[0].UserID, "user-42")
	}
	if logs[0].Action != db.AuditActionBookCreated {
		t.Errorf("action = %q, want %q", logs[0].Action, db.AuditActionBookCreated)
	}
	if logs[0].EntityType != "book" {
		t.Errorf("entity type = %q, want %q", logs[0].EntityType, "book")
	}
	if logs[0].EntityID != "book-1" {
		t.Errorf("entity id = %q, want %q", logs[0].EntityID, "book-1")
	}
	if logs[0].Metadata == nil {
		t.Fatal("metadata = nil, want JSON metadata")
	}
}
