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
	require.NotEqual(t, "", e.ID)
	require.NotNil(t, e.UserID)
	require.Equal(t, "user1", *e.UserID)
	require.Equal(t, AuditActionLibraryCreated, e.Action)
	require.Equal(t, "library", e.EntityType)
	require.Equal(t, "lib1", e.EntityID)
	require.NotNil(t, e.Metadata)
	require.False(t, e.CreatedAt.IsZero())
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
	require.Nil(t, entries[0].UserID)
	require.Nil(t, entries[0].Metadata)
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
	require.Equal(t, 5, total2)
	require.Len(t, entries, 2)

	entries2, _, err := d.ListAuditLogs(ctx, 2, 2)
	require.NoError(t, err, "ListAuditLogs() page2 error")
	require.Len(t, entries2, 2)

	entries3, _, err := d.ListAuditLogs(ctx, 2, 4)
	require.NoError(t, err, "ListAuditLogs() page3 error")
	require.Len(t, entries3, 1)
}

func TestListAuditLogs_Empty(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	entries, total, err := d.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "ListAuditLogs() error")
	require.Equal(t, 0, total)
	require.Len(t, entries, 0)
}

// TestListAuditLogs_OffsetBeyondTotal verifies that total is correct even when
// offset exceeds the number of rows (the window-function count fallback path).
func TestListAuditLogs_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	for range 3 {
		require.NoError(t, d.CreateAuditLog(ctx, "user1", AuditActionBookCreated, "book", "book-x", nil))
	}

	entries, total, err := d.ListAuditLogs(ctx, 10, 100)
	require.NoError(t, err, "ListAuditLogs() error")
	require.Equal(t, 3, total, "total should reflect real count even beyond offset")
	require.Len(t, entries, 0)
}
