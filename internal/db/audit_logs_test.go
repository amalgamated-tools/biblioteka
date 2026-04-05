package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAuditLog(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	err := d.CreateAuditLog(ctx, "user1", AuditActionLibraryCreated, "library", "lib1", map[string]any{"name": "Fiction"})
	require.NoError(t, err, "CreateAuditLog() error")

	entries, total, err := d.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "ListAuditLogs() error")
	require.Equal(t, 1, total)
	require.Len(t, entries, 1)

	e := entries[0]
	if e.ID == "" {
		t.Error("ID is empty")
	}
	if e.UserID == nil || *e.UserID != "user1" {
		t.Errorf("UserID = %v, want \"user1\"", e.UserID)
	}
	if e.Action != AuditActionLibraryCreated {
		t.Errorf("Action = %q, want %q", e.Action, AuditActionLibraryCreated)
	}
	if e.EntityType != "library" {
		t.Errorf("EntityType = %q, want %q", e.EntityType, "library")
	}
	if e.EntityID != "lib1" {
		t.Errorf("EntityID = %q, want %q", e.EntityID, "lib1")
	}
	if e.Metadata == nil {
		t.Error("Metadata is nil, want JSON string")
	}
	if e.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestCreateAuditLog_SystemAction(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	// Empty userID → NULL user_id in DB.
	err := d.CreateAuditLog(ctx, "", AuditActionBookCreated, "book", "book1", nil)
	require.NoError(t, err, "CreateAuditLog() error")

	entries, _, err := d.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "ListAuditLogs() error")
	require.NotEmpty(t, entries)
	if entries[0].UserID != nil {
		t.Errorf("UserID = %v, want nil", entries[0].UserID)
	}
	if entries[0].Metadata != nil {
		t.Errorf("Metadata = %v, want nil", entries[0].Metadata)
	}
}

func TestListAuditLogs_Pagination(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	for i := range 5 {
		_ = i
		err := d.CreateAuditLog(ctx, "user1", AuditActionBookCreated, "book", "book-x", nil)
		require.NoError(t, err, "CreateAuditLog() error")
	}

	_, total, err := d.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "ListAuditLogs() error")
	require.Equal(t, 5, total)

	entries, total2, err := d.ListAuditLogs(ctx, 2, 0)
	require.NoError(t, err, "ListAuditLogs() page1 error")
	if total2 != 5 {
		t.Errorf("total2 = %d, want 5", total2)
	}
	if len(entries) != 2 {
		t.Errorf("page1 len = %d, want 2", len(entries))
	}

	entries2, _, err := d.ListAuditLogs(ctx, 2, 2)
	require.NoError(t, err, "ListAuditLogs() page2 error")
	if len(entries2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(entries2))
	}

	entries3, _, err := d.ListAuditLogs(ctx, 2, 4)
	require.NoError(t, err, "ListAuditLogs() page3 error")
	if len(entries3) != 1 {
		t.Errorf("page3 len = %d, want 1", len(entries3))
	}
}

func TestListAuditLogs_Empty(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	entries, total, err := d.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "ListAuditLogs() error")
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}
