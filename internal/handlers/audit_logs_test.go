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

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	// CreateUser itself may generate audit entries; just verify we got at least 2 from our inserts.
	require.GreaterOrEqual(t, len(resp.Entries), 2, "expected at least 2 entries")
	require.GreaterOrEqual(t, resp.Total, 2, "expected total >= 2")
}

func TestHandleAuditLogs_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleAuditLogs_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/audit-logs", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleAuditLogs_DefaultPagination(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, 50, resp.Limit)
	require.Equal(t, 0, resp.Offset)
}

func TestHandleAuditLogs_CustomPagination(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=10&offset=5", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, 10, resp.Limit)
	require.Equal(t, 5, resp.Offset)
}

func TestHandleAuditLogs_LimitCappedAtMax(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=999", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, 200, resp.Limit)
}

func TestHandleAuditLogs_InvalidLimitFallsBackToDefault(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=abc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, 50, resp.Limit, "invalid limit should fall back to default")
}

func TestHandleAuditLogs_InvalidOffsetFallsBackToDefault(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?offset=abc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, 0, resp.Offset, "invalid offset should fall back to default")
}

func TestHandleAuditLogs_NegativeOffsetFallsBackToDefault(t *testing.T) {
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?offset=-1", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, 0, resp.Offset, "negative offset should fall back to default")
}

func TestHandleAuditLogs_EmptyList(t *testing.T) {
	// Use a fresh DB with only the admin user and set a high offset
	// to guarantee no entries are returned.
	h, adminID, _ := setupAuditLogHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/audit-logs?offset=10000", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleAuditLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Len(t, resp.Entries, 0)
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

	require.Equal(t, http.StatusOK, w.Code)

	var resp auditLogListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	// Find the entry we inserted with metadata.
	var found bool
	for _, e := range resp.Entries {
		if e.EntityID == "book-meta-1" {
			found = true
			require.NotNil(t, e.Metadata)
			var m map[string]any
			require.NoError(t, json.Unmarshal(e.Metadata, &m), "unmarshal metadata")
			require.Equal(t, "Go Programming", m["title"])
			require.Equal(t, "978-0-123", m["isbn"])
			break
		}
	}
	require.True(t, found)
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

	require.Equal(t, "log-1", dto.ID)
	require.Nil(t, dto.Metadata)
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

	require.Nil(t, dto.Metadata)
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

	require.NotNil(t, dto.Metadata)

	var m map[string]any
	require.NoError(t, json.Unmarshal(dto.Metadata, &m), "unmarshal metadata")
	require.Equal(t, "Test Book", m["title"])
	require.Equal(t, float64(42), m["pages"])
	require.Equal(t, db.AuditActionBookCreated, dto.Action)
	require.Equal(t, "book", dto.EntityType)
	require.Equal(t, "book-3", dto.EntityID)
}
