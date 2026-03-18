package db

import (
	"context"
	"testing"
)

func TestCreateAuditLog(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	err := d.CreateAuditLog(ctx, "user1", AuditActionLibraryCreated, "library", "lib1", map[string]any{"name": "Fiction"})
	if err != nil {
		failNowf(t, "CreateAuditLog() error: %v", err)
	}

	entries, total, err := d.ListAuditLogs(ctx, 10, 0)
	if err != nil {
		failNowf(t, "ListAuditLogs() error: %v", err)
	}
	if total != 1 {
		failNowf(t, "total = %d, want 1", total)
	}
	if len(entries) != 1 {
		failNowf(t, "len(entries) = %d, want 1", len(entries))
	}

	e := entries[0]
	if e.ID == "" {
		fail(t, "ID is empty")
	}
	if e.UserID == nil || *e.UserID != "user1" {
		failf(t, "UserID = %v, want \"user1\"", e.UserID)
	}
	if e.Action != AuditActionLibraryCreated {
		failf(t, "Action = %q, want %q", e.Action, AuditActionLibraryCreated)
	}
	if e.EntityType != "library" {
		failf(t, "EntityType = %q, want %q", e.EntityType, "library")
	}
	if e.EntityID != "lib1" {
		failf(t, "EntityID = %q, want %q", e.EntityID, "lib1")
	}
	if e.Metadata == nil {
		fail(t, "Metadata is nil, want JSON string")
	}
	if e.CreatedAt.IsZero() {
		fail(t, "CreatedAt is zero")
	}
}

func TestCreateAuditLog_SystemAction(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	// Empty userID → NULL user_id in DB.
	err := d.CreateAuditLog(ctx, "", AuditActionBookCreated, "book", "book1", nil)
	if err != nil {
		failNowf(t, "CreateAuditLog() error: %v", err)
	}

	entries, _, err := d.ListAuditLogs(ctx, 10, 0)
	if err != nil {
		failNowf(t, "ListAuditLogs() error: %v", err)
	}
	if len(entries) == 0 {
		failNow(t, "expected at least one entry")
	}
	if entries[0].UserID != nil {
		failf(t, "UserID = %v, want nil", entries[0].UserID)
	}
	if entries[0].Metadata != nil {
		failf(t, "Metadata = %v, want nil", entries[0].Metadata)
	}
}

func TestListAuditLogs_Pagination(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	for i := range 5 {
		_ = i
		err := d.CreateAuditLog(ctx, "user1", AuditActionBookCreated, "book", "book-x", nil)
		if err != nil {
			failNowf(t, "CreateAuditLog() error: %v", err)
		}
	}

	_, total, err := d.ListAuditLogs(ctx, 10, 0)
	if err != nil {
		failNowf(t, "ListAuditLogs() error: %v", err)
	}
	if total != 5 {
		failNowf(t, "total = %d, want 5", total)
	}

	entries, total2, err := d.ListAuditLogs(ctx, 2, 0)
	if err != nil {
		failNowf(t, "ListAuditLogs() page1 error: %v", err)
	}
	if total2 != 5 {
		failf(t, "total2 = %d, want 5", total2)
	}
	if len(entries) != 2 {
		failf(t, "page1 len = %d, want 2", len(entries))
	}

	entries2, _, err := d.ListAuditLogs(ctx, 2, 2)
	if err != nil {
		failNowf(t, "ListAuditLogs() page2 error: %v", err)
	}
	if len(entries2) != 2 {
		failf(t, "page2 len = %d, want 2", len(entries2))
	}

	entries3, _, err := d.ListAuditLogs(ctx, 2, 4)
	if err != nil {
		failNowf(t, "ListAuditLogs() page3 error: %v", err)
	}
	if len(entries3) != 1 {
		failf(t, "page3 len = %d, want 1", len(entries3))
	}
}

func TestListAuditLogs_Empty(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	entries, total, err := d.ListAuditLogs(ctx, 10, 0)
	if err != nil {
		failNowf(t, "ListAuditLogs() error: %v", err)
	}
	if total != 0 {
		failf(t, "total = %d, want 0", total)
	}
	if len(entries) != 0 {
		failf(t, "len(entries) = %d, want 0", len(entries))
	}
}
