package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

// mustMarshal marshals v to JSON and fails the test if marshaling fails.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err, "mustMarshal")
	return data
}

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
		require.NoError(t, err, "failed to unmarshal")
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
		require.NoError(t, err, "failed to unmarshal")
	}
	if result["error"] != "something went wrong" {
		t.Errorf("error = %q, want %q", result["error"], "something went wrong")
	}
}

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
			if got != tt.wantValid {
				require.Failf(t, "failed", "validateName(%q) = %v, want %v", tt.input, got, tt.wantValid)
			}
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
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				require.NoError(t, err, "failed to unmarshal")
			}
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
				require.Failf(t, "failed", "handleNameErr() = %v, want %v", got, tt.wantHandled)
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
				require.NoError(t, err, "failed to unmarshal")
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
				require.Failf(t, "failed", "handleDBErr() = %v, want %v", got, tt.wantHandled)
			}
			if !tt.wantHandled {
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			var result map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				require.NoError(t, err, "failed to unmarshal")
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
	require.NoError(t, err, "list audit logs")
	if len(logs) != 1 {
		require.Failf(t, "failed", "len(logs) = %d, want 1", len(logs))
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
		require.Fail(t, "metadata = nil, want JSON metadata")
	}
}

func Test_HandleUpdateErr(t *testing.T) {
	var (
		errInvalid = errors.New("invalid name")
		errExists  = errors.New("name exists")
		errOther   = errors.New("other error")
	)

	tests := []struct {
		name        string
		err         error
		resourceArt string
		resource    string
		id          string
		wantHandled bool
		wantCode    int
		wantErrMsg  string
	}{
		{
			name:        "nil error not handled",
			err:         nil,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-1",
			wantHandled: false,
		},
		{
			name:        "not found yields 404",
			err:         sql.ErrNoRows,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-1",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantErrMsg:  "author not found",
		},
		{
			name:        "wrapped not found yields 404",
			err:         fmt.Errorf("db: %w", sql.ErrNoRows),
			resourceArt: "a series",
			resource:    "series",
			id:          "ser-1",
			wantHandled: true,
			wantCode:    http.StatusNotFound,
			wantErrMsg:  "series not found",
		},
		{
			name:        "invalid name yields 400",
			err:         errInvalid,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-2",
			wantHandled: true,
			wantCode:    http.StatusBadRequest,
			wantErrMsg:  "name is required",
		},
		{
			name:        "duplicate name yields 409",
			err:         errExists,
			resourceArt: "a series",
			resource:    "series",
			id:          "ser-2",
			wantHandled: true,
			wantCode:    http.StatusConflict,
			wantErrMsg:  "a series with that name already exists",
		},
		{
			name:        "other error yields 500",
			err:         errOther,
			resourceArt: "an author",
			resource:    "author",
			id:          "auth-3",
			wantHandled: true,
			wantCode:    http.StatusInternalServerError,
			wantErrMsg:  "failed to update author",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := handleUpdateErr(t.Context(), w, tt.err, errInvalid, errExists, tt.resourceArt, tt.resource, tt.id)
			if got != tt.wantHandled {
				require.Failf(t, "failed", "handleUpdateErr() = %v, want %v", got, tt.wantHandled)
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
				require.NoError(t, err, "failed to unmarshal")
			}
			if result["error"] != tt.wantErrMsg {
				t.Errorf("error = %q, want %q", result["error"], tt.wantErrMsg)
			}
		})
	}
}

func Test_ListEntities(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string `json:"label"`
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}

	t.Run("error yields 500", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return nil, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var result map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			require.NoError(t, err, "failed to unmarshal")
		}
		if result["error"] != "failed to list widgets" {
			t.Errorf("error = %q, want %q", result["error"], "failed to list widgets")
		}
	})

	t.Run("success converts to DTOs", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
			require.NoError(t, err, "failed to unmarshal")
		}
		if len(dtos) != 2 {
			require.Failf(t, "failed", "len = %d, want 2", len(dtos))
		}
		if dtos[0].Label != "Alpha" {
			t.Errorf("dtos[0].Label = %q, want %q", dtos[0].Label, "Alpha")
		}
		if dtos[1].Label != "Beta" {
			t.Errorf("dtos[1].Label = %q, want %q", dtos[1].Label, "Beta")
		}
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		listFn := func(_ context.Context) ([]entity, error) {
			return []entity{}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		listEntities(w, r, "widgets", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
			require.NoError(t, err, "failed to unmarshal")
		}
		if len(dtos) != 0 {
			t.Errorf("len = %d, want 0", len(dtos))
		}
	})
}

func Test_MapSlice(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}

	t.Run("converts elements", func(t *testing.T) {
		items := []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}
		result := mapSlice(items, toDTO)
		if len(result) != 2 {
			require.Failf(t, "failed", "len = %d, want 2", len(result))
		}
		if result[0].Label != "Alpha" {
			t.Errorf("result[0].Label = %q, want %q", result[0].Label, "Alpha")
		}
		if result[1].Label != "Beta" {
			t.Errorf("result[1].Label = %q, want %q", result[1].Label, "Beta")
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := mapSlice([]entity{}, toDTO)
		if result == nil {
			require.Fail(t, "result is nil, want non-nil empty slice")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("nil input returns empty slice", func(t *testing.T) {
		result := mapSlice(nil, toDTO)
		if result == nil {
			require.Fail(t, "result is nil, want non-nil empty slice")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})
}

func TestRequestScheme(t *testing.T) {
	tests := []struct {
		name   string
		tls    bool
		header string // X-Forwarded-Proto value
		want   string
	}{
		{name: "plain HTTP", tls: false, header: "", want: "http"},
		{name: "TLS connection", tls: true, header: "", want: "https"},
		{name: "forwarded https", tls: false, header: "https", want: "https"},
		{name: "forwarded http", tls: true, header: "http", want: "http"},
		{name: "uppercase HTTPS", tls: false, header: "HTTPS", want: "https"},
		{name: "mixed case Https", tls: false, header: "Https", want: "https"},
		{name: "padded whitespace", tls: false, header: "  https  ", want: "https"},
		{name: "invalid proto ignored", tls: false, header: "javascript:", want: "http"},
		{name: "ftp proto ignored", tls: false, header: "ftp", want: "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			if tt.header != "" {
				r.Header.Set("X-Forwarded-Proto", tt.header)
			}
			if got := requestScheme(r); got != tt.want {
				t.Errorf("requestScheme() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_ListUserEntities(t *testing.T) {
	type entity struct {
		ID   int
		Name string
	}
	type dto struct {
		Label string `json:"label"`
	}
	toDTO := func(e *entity) dto {
		return dto{Label: e.Name}
	}

	t.Run("error yields 500", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return nil, errors.New("db failure")
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var result map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			require.NoError(t, err, "failed to unmarshal")
		}
		if result["error"] != "failed to list tokens" {
			t.Errorf("error = %q, want %q", result["error"], "failed to list tokens")
		}
	})

	t.Run("passes user ID to list function", func(t *testing.T) {
		var capturedUserID string
		listFn := func(_ context.Context, userID string) ([]entity, error) {
			capturedUserID = userID
			return []entity{{ID: 1, Name: "Alpha"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-42")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if capturedUserID != "user-42" {
			t.Errorf("capturedUserID = %q, want %q", capturedUserID, "user-42")
		}
	})

	t.Run("nil slice returns empty JSON array", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return nil, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
			require.NoError(t, err, "failed to unmarshal")
		}
		if len(dtos) != 0 {
			t.Errorf("len = %d, want 0", len(dtos))
		}
	})

	t.Run("success converts to DTOs", func(t *testing.T) {
		listFn := func(_ context.Context, _ string) ([]entity, error) {
			return []entity{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}, nil
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = withUserID(r, "user-1")
		listUserEntities(w, r, "tokens", listFn, toDTO)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var dtos []dto
		if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
			require.NoError(t, err, "failed to unmarshal")
		}
		if len(dtos) != 2 {
			require.Failf(t, "failed", "len = %d, want 2", len(dtos))
		}
		if dtos[0].Label != "Alpha" {
			t.Errorf("dtos[0].Label = %q, want %q", dtos[0].Label, "Alpha")
		}
		if dtos[1].Label != "Beta" {
			t.Errorf("dtos[1].Label = %q, want %q", dtos[1].Label, "Beta")
		}
	})
}

func Test_HandleTokenCreate(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Token User", "tokens@example.com", "password1")
	require.NoError(t, err, "create user")

	t.Run("empty name yields 400", func(t *testing.T) {
		ops := tokenOps{
			db:              d,
			resource:        "test token",
			auditEntityType: "test_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				require.Fail(t, "create should not be called for invalid name")
				return "", nil, nil
			},
		}

		body := mustMarshal(t, map[string]string{"name": ""})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("create error yields 500", func(t *testing.T) {
		d := newTestDB(t)
		user, err := d.CreateUser(t.Context(), "Error User", "error@example.com", "password1")
		require.NoError(t, err, "create user")

		ops := tokenOps{
			db:              d,
			resource:        "test token",
			auditEntityType: "test_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "", nil, errors.New("db failure")
			},
		}

		body := mustMarshal(t, map[string]string{"name": "My Token"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}

		// Verify no audit log written on error.
		logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
		require.NoError(t, err, "list audit logs")
		if len(logs) != 0 {
			t.Errorf("expected no audit logs on error, got %d", len(logs))
		}
	})

	t.Run("success returns 201 with no-store headers", func(t *testing.T) {
		type tokenResp struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}
		ops := tokenOps{
			db:              d,
			resource:        "test token",
			auditEntityType: "test_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "entity-123", tokenResp{ID: "entity-123", Token: "secret"}, nil
			},
		}

		body := mustMarshal(t, map[string]string{"name": "My Token"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
		}
		var resp tokenResp
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			require.NoError(t, err, "unmarshal")
		}
		if resp.Token != "secret" {
			t.Errorf("token = %q, want %q", resp.Token, "secret")
		}
	})

	t.Run("success writes audit log", func(t *testing.T) {
		ops := tokenOps{
			db:              d,
			resource:        "audit token",
			auditEntityType: "audit_token",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "audit-entity-1", map[string]string{"token": "val"}, nil
			},
		}

		body := mustMarshal(t, map[string]string{"name": "Audited"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusCreated {
			require.Failf(t, "failed", "status = %d, want %d", w.Code, http.StatusCreated)
		}

		logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
		require.NoError(t, err, "list audit logs")
		found := false
		for _, l := range logs {
			if l.Action == db.AuditActionAPIKeyCreated && l.EntityID == "audit-entity-1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected audit log entry for created token")
		}
	})

	t.Run("tokenError returns specific message", func(t *testing.T) {
		d := newTestDB(t)
		user, err := d.CreateUser(t.Context(), "Test User", "test3@example.com", "password1")
		require.NoError(t, err, "create user")

		ops := tokenOps{
			db:              d,
			resource:        "widget",
			auditEntityType: "widget",
			auditCreate:     db.AuditActionAPIKeyCreated,
			create: func(_ context.Context, _, _ string) (string, any, error) {
				return "", nil, &tokenError{err: errors.New("rng failure"), message: "failed to generate widget"}
			},
		}

		body := mustMarshal(t, map[string]string{"name": "My Widget"})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		handleTokenCreate(ops, w, r)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var result map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			require.NoError(t, err, "unmarshal")
		}
		if result["error"] != "failed to generate widget" {
			t.Errorf("error = %q, want %q", result["error"], "failed to generate widget")
		}
	})
}
