package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateGoodreadsMetadata(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Project Hail Mary"
	authorName := "Andy Weir"
	isbn13 := "9780593135204"
	grID := "kca://book/amzn1.gr.book.v1.def456"
	bookLegacyID := int64(54493401)

	gm, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title, nil, nil, nil, &isbn13, &grID, nil, nil, nil, nil, nil, nil,
		&authorName, nil, nil, nil,
		&bookLegacyID, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")
	if gm.ID == "" {
		t.Error("ID is empty")
	}
	if gm.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", gm.UserID, user.ID)
	}
	if gm.Status != GoodreadsMetadataStatusPending {
		t.Errorf("Status = %q, want %q", gm.Status, GoodreadsMetadataStatusPending)
	}
	if gm.Title == nil || *gm.Title != title {
		t.Errorf("Title = %v, want %q", gm.Title, title)
	}
	if gm.AuthorName == nil || *gm.AuthorName != authorName {
		t.Errorf("AuthorName = %v, want %q", gm.AuthorName, authorName)
	}
	if gm.ISBN13 == nil || *gm.ISBN13 != isbn13 {
		t.Errorf("ISBN13 = %v, want %q", gm.ISBN13, isbn13)
	}
	if gm.GoodreadsBookLegacyID == nil || *gm.GoodreadsBookLegacyID != bookLegacyID {
		t.Errorf("GoodreadsBookLegacyID = %v, want %d", gm.GoodreadsBookLegacyID, bookLegacyID)
	}
	if gm.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestGetGoodreadsMetadata(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	found, err := d.GetGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.NoError(t, err, "GetGoodreadsMetadata() error")
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Title == nil || *found.Title != title {
		t.Errorf("Title = %v, want %q", found.Title, title)
	}
}

func TestGetGoodreadsMetadata_NotFound(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	_, err = d.GetGoodreadsMetadata(t.Context(), user.ID, "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetGoodreadsMetadata_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1, err := d.CreateUser(t.Context(), "User One", "user1@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")
	user2, err := d.CreateUser(t.Context(), "User Two", "user2@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user1.ID,
		nil, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	_, err = d.GetGoodreadsMetadata(t.Context(), user2.ID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for wrong user, got %v", err)
	}
}

func TestListGoodreadsMetadataByUser(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title1 := "Book One"
	title2 := "Book Two"
	_, err = d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title1, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")
	_, err = d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title2, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	results, err := d.ListGoodreadsMetadataByUser(t.Context(), user.ID, 50, 0)
	require.NoError(t, err, "ListGoodreadsMetadataByUser() error")
	if len(results) != 2 {
		require.Failf(t, "failed", "expected 2 results, got %d", len(results))
	}
}

func TestListGoodreadsMetadataByStatus(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title1 := "Pending Book"
	gm1, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title1, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	// Update status of one to applied
	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, gm1.ID, GoodreadsMetadataStatusApplied)
	require.NoError(t, err, "UpdateGoodreadsMetadataStatus() error")

	title2 := "Still Pending"
	_, err = d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title2, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	pending, err := d.ListGoodreadsMetadataByStatus(t.Context(), user.ID, GoodreadsMetadataStatusPending, 50, 0)
	require.NoError(t, err, "ListGoodreadsMetadataByStatus() error")
	if len(pending) != 1 {
		require.Failf(t, "failed", "expected 1 pending, got %d", len(pending))
	}
	if pending[0].Title == nil || *pending[0].Title != title2 {
		t.Errorf("Title = %v, want %q", pending[0].Title, title2)
	}

	applied, err := d.ListGoodreadsMetadataByStatus(t.Context(), user.ID, GoodreadsMetadataStatusApplied, 50, 0)
	require.NoError(t, err, "ListGoodreadsMetadataByStatus() error")
	if len(applied) != 1 {
		require.Failf(t, "failed", "expected 1 applied, got %d", len(applied))
	}
}

func TestUpdateGoodreadsMetadataStatus(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	updated, err := d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, created.ID, GoodreadsMetadataStatusRejected)
	require.NoError(t, err, "UpdateGoodreadsMetadataStatus() error")
	if updated.Status != GoodreadsMetadataStatusRejected {
		t.Errorf("Status = %q, want %q", updated.Status, GoodreadsMetadataStatusRejected)
	}

	// Attempt to set an invalid status and ensure it fails without changing the row.
	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, created.ID, "invalid")
	require.Error(t, err, "UpdateGoodreadsMetadataStatus() with invalid status expected error, got nil")

	// Verify that the status in the database remains unchanged after the failed update.
	fetched, err := d.GetGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.NoError(t, err, "GetGoodreadsMetadata() error after invalid status update")
	if fetched.Status != updated.Status {
		t.Errorf("Status changed after invalid status update: got %q, want %q", fetched.Status, updated.Status)
	}
}

func TestUpdateGoodreadsMetadataStatus_InvalidStatus(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, created.ID, "oops")
	require.Error(t, err, "expected error for invalid status, got nil")
	if !errors.Is(err, ErrInvalidGoodreadsMetadataStatus) {
		t.Errorf("expected ErrInvalidGoodreadsMetadataStatus, got %v", err)
	}
}

func TestDeleteGoodreadsMetadata(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		nil, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	err = d.DeleteGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.NoError(t, err, "DeleteGoodreadsMetadata() error")

	_, err = d.GetGoodreadsMetadata(t.Context(), user.ID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteGoodreadsMetadata_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1, err := d.CreateUser(t.Context(), "User One", "user1@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")
	user2, err := d.CreateUser(t.Context(), "User Two", "user2@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user1.ID,
		nil, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	err = d.DeleteGoodreadsMetadata(t.Context(), user2.ID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for wrong user, got %v", err)
	}

	// Verify it still exists for the original user
	_, err = d.GetGoodreadsMetadata(t.Context(), user1.ID, created.ID)
	if err != nil {
		t.Errorf("expected row to still exist for original user, got %v", err)
	}
}

func TestCreateGoodreadsMetadata_WithBookID(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	book, err := d.CreateBook(t.Context(), "Existing Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")

	title := "Updated Metadata"
	gm, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		&book.ID, &title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")
	if gm.BookID == nil || *gm.BookID != book.ID {
		t.Errorf("BookID = %v, want %q", gm.BookID, book.ID)
	}
}
