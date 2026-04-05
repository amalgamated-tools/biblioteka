package handlers

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// mustMarshal marshals v to JSON and fails the test if marshaling fails.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
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
