package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

// setupAuditLogHandler creates a DB with an admin user (first user) and a regular user,
// and returns a handler, the admin ID, and the regular user ID.
func setupAuditLogHandler(t *testing.T) (*AuditLogHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	h := &AuditLogHandler{DB: d}

	admin, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	require.NoError(t, err, "create admin")
	regular, err := d.CreateUser(t.Context(), "Regular", "regular@example.com", "password1")
	require.NoError(t, err, "create regular user")
	return h, admin.ID, regular.ID
}

func TestHandleAuditLogs_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	// Seed a couple of audit log entries.
	ctx := t.Context()
	require.NoError(t, h.DB.CreateAuditLog(ctx, adminID, db.AuditActionBookCreated, "book", "book-1", nil), "create audit log")
	require.NoError(t, h.DB.CreateAuditLog(ctx, adminID, db.AuditActionLibraryCreated, "library", "lib-1", nil), "create audit log")

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	// CreateUser itself may generate audit entries; just verify we got at least 2 from our inserts.
	if len(resp.Entries) < 2 {
		t.Errorf("expected at least 2 entries, got %d", len(resp.Entries))
	}
	if resp.Total < 2 {
		t.Errorf("expected total >= 2, got %d", resp.Total)
	}
}

func TestHandleAuditLogs_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleAuditLogs_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/audit-logs", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAuditLogs_DefaultPagination(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want 50", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("offset = %d, want 0", resp.Offset)
	}
}

func TestHandleAuditLogs_CustomPagination(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=10&offset=5", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Limit != 10 {
		t.Errorf("limit = %d, want 10", resp.Limit)
	}
	if resp.Offset != 5 {
		t.Errorf("offset = %d, want 5", resp.Offset)
	}
}

func TestHandleAuditLogs_LimitCappedAtMax(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=999", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Limit != 200 {
		t.Errorf("limit = %d, want 200 (max)", resp.Limit)
	}
}

func TestHandleAuditLogs_InvalidLimit(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=abc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleAuditLogs_InvalidOffset(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?offset=abc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleAuditLogs_NegativeOffset(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?offset=-1", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleAuditLogs_EmptyList(t *testing.T) {
	// Use a fresh DB with only the admin user and set a high offset
	// to guarantee no entries are returned.
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?offset=10000", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}

func TestHandleAuditLogs_WithMetadata(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	meta := map[string]any{"title": "Go Programming", "isbn": "978-0-123"}
	ctx := t.Context()
	require.NoError(t, h.DB.CreateAuditLog(ctx, adminID, db.AuditActionBookCreated, "book", "book-meta-1", meta), "create audit log with metadata")

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	// Find the entry we inserted with metadata.
	var found bool
	for _, e := range resp.Entries {
		if e.EntityID == "book-meta-1" {
			found = true
			if e.Metadata == nil {
				require.Fail(t, "expected metadata to be non-nil")
			}
			var m map[string]any
			require.NoError(t, json.Unmarshal(e.Metadata, &m), "unmarshal metadata")
			if m["title"] != "Go Programming" {
				t.Errorf("metadata title = %v, want %q", m["title"], "Go Programming")
			}
			if m["isbn"] != "978-0-123" {
				t.Errorf("metadata isbn = %v, want %q", m["isbn"], "978-0-123")
			}
			break
		}
	}
	if !found {
		t.Error("audit log entry with entity_id=book-meta-1 not found")
	}
}

func TestToAuditLogDTO_NilMetadata(t *testing.T) {
	entry := &db.AuditLog{
		ID:         "log-1",
		UserID:     new("user-1"),
		Action:     db.AuditActionBookCreated,
		EntityType: "book",
		EntityID:   "book-1",
		Metadata:   nil,
	}

	dto := toAuditLogDTO(entry)

	if dto.ID != "log-1" {
		t.Errorf("ID = %q, want %q", dto.ID, "log-1")
	}
	if dto.Metadata != nil {
		t.Errorf("Metadata = %s, want nil", string(dto.Metadata))
	}
}

func TestToAuditLogDTO_EmptyMetadata(t *testing.T) {
	empty := ""
	entry := &db.AuditLog{
		ID:         "log-2",
		UserID:     new("user-1"),
		Action:     db.AuditActionBookUpdated,
		EntityType: "book",
		EntityID:   "book-2",
		Metadata:   &empty,
	}

	dto := toAuditLogDTO(entry)

	if dto.Metadata != nil {
		t.Errorf("Metadata = %s, want nil for empty string", string(dto.Metadata))
	}
}

func TestToAuditLogDTO_ValidMetadata(t *testing.T) {
	meta := `{"title":"Test Book","pages":42}`
	entry := &db.AuditLog{
		ID:         "log-3",
		UserID:     new("user-1"),
		Action:     db.AuditActionBookCreated,
		EntityType: "book",
		EntityID:   "book-3",
		Metadata:   &meta,
	}

	dto := toAuditLogDTO(entry)

	if dto.Metadata == nil {
		require.Fail(t, "Metadata should not be nil")
	}

	var m map[string]any
	require.NoError(t, json.Unmarshal(dto.Metadata, &m), "unmarshal metadata")
	if m["title"] != "Test Book" {
		t.Errorf("metadata title = %v, want %q", m["title"], "Test Book")
	}
	if m["pages"] != float64(42) {
		t.Errorf("metadata pages = %v, want 42", m["pages"])
	}
	if dto.Action != db.AuditActionBookCreated {
		t.Errorf("Action = %q, want %q", dto.Action, db.AuditActionBookCreated)
	}
	if dto.EntityType != "book" {
		t.Errorf("EntityType = %q, want %q", dto.EntityType, "book")
	}
	if dto.EntityID != "book-3" {
		t.Errorf("EntityID = %q, want %q", dto.EntityID, "book-3")
	}
}
