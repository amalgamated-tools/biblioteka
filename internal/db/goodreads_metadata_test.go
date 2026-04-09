package db

import (
	"database/sql"
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
		GoodreadsMetadataInput{
			Title:                 &title,
			ISBN13:                &isbn13,
			GoodreadsID:           &grID,
			AuthorName:            &authorName,
			GoodreadsBookLegacyID: &bookLegacyID,
		},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")
	require.NotEqual(t, "", gm.ID)
	require.Equal(t, user.ID, gm.UserID)
	require.Equal(t, GoodreadsMetadataStatusPending, gm.Status)
	require.NotNil(t, gm.Title)
	require.Equal(t, title, *gm.Title)
	require.NotNil(t, gm.AuthorName)
	require.Equal(t, authorName, *gm.AuthorName)
	require.NotNil(t, gm.ISBN13)
	require.Equal(t, isbn13, *gm.ISBN13)
	require.NotNil(t, gm.GoodreadsBookLegacyID)
	require.Equal(t, bookLegacyID, *gm.GoodreadsBookLegacyID)
	require.False(t, gm.CreatedAt.IsZero())
}

func TestGetGoodreadsMetadata(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	found, err := d.GetGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.NoError(t, err, "GetGoodreadsMetadata() error")
	require.Equal(t, created.ID, found.ID)
	require.NotNil(t, found.Title)
	require.Equal(t, title, *found.Title)
}

func TestGetGoodreadsMetadata_NotFound(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	_, err = d.GetGoodreadsMetadata(t.Context(), user.ID, "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
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
		GoodreadsMetadataInput{Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	_, err = d.GetGoodreadsMetadata(t.Context(), user2.ID, created.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListGoodreadsMetadataByUser(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title1 := "Book One"
	title2 := "Book Two"
	_, err = d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title1},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")
	_, err = d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title2},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	results, err := d.ListGoodreadsMetadataByUser(t.Context(), user.ID, 50, 0)
	require.NoError(t, err, "ListGoodreadsMetadataByUser() error")
	require.Len(t, results, 2)
}

func TestListGoodreadsMetadataByStatus(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title1 := "Pending Book"
	gm1, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title1},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	// Update status of one to applied
	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, gm1.ID, GoodreadsMetadataStatusApplied)
	require.NoError(t, err, "UpdateGoodreadsMetadataStatus() error")

	title2 := "Still Pending"
	_, err = d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title2},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	pending, err := d.ListGoodreadsMetadataByStatus(t.Context(), user.ID, GoodreadsMetadataStatusPending, 50, 0)
	require.NoError(t, err, "ListGoodreadsMetadataByStatus() error")
	require.Len(t, pending, 1)
	require.NotNil(t, pending[0].Title)
	require.Equal(t, title2, *pending[0].Title)

	applied, err := d.ListGoodreadsMetadataByStatus(t.Context(), user.ID, GoodreadsMetadataStatusApplied, 50, 0)
	require.NoError(t, err, "ListGoodreadsMetadataByStatus() error")
	require.Len(t, applied, 1)
}

func TestUpdateGoodreadsMetadataStatus(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	updated, err := d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, created.ID, GoodreadsMetadataStatusRejected)
	require.NoError(t, err, "UpdateGoodreadsMetadataStatus() error")
	require.Equal(t, GoodreadsMetadataStatusRejected, updated.Status)

	// Attempt to set an invalid status and ensure it fails without changing the row.
	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, created.ID, "invalid")
	require.Error(t, err, "UpdateGoodreadsMetadataStatus() with invalid status expected error, got nil")

	// Verify that the status in the database remains unchanged after the failed update.
	fetched, err := d.GetGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.NoError(t, err, "GetGoodreadsMetadata() error after invalid status update")
	require.Equal(t, updated.Status, fetched.Status)
}

func TestUpdateGoodreadsMetadataStatus_InvalidStatus(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, created.ID, "oops")
	require.Error(t, err, "expected error for invalid status, got nil")
	require.ErrorIs(t, err, ErrInvalidGoodreadsMetadataStatus)
}

func TestDeleteGoodreadsMetadata(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	title := "Test Book"
	created, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	err = d.DeleteGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.NoError(t, err, "DeleteGoodreadsMetadata() error")

	_, err = d.GetGoodreadsMetadata(t.Context(), user.ID, created.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
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
		GoodreadsMetadataInput{Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")

	err = d.DeleteGoodreadsMetadata(t.Context(), user2.ID, created.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Verify it still exists for the original user
	_, err = d.GetGoodreadsMetadata(t.Context(), user1.ID, created.ID)
	require.NoError(t, err)
}

func TestCreateGoodreadsMetadata_WithBookID(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err, "CreateUser() error")

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Existing Book"})
	require.NoError(t, err, "CreateBook() error")

	title := "Updated Metadata"
	gm, err := d.CreateGoodreadsMetadata(
		t.Context(), user.ID,
		GoodreadsMetadataInput{BookID: &book.ID, Title: &title},
	)
	require.NoError(t, err, "CreateGoodreadsMetadata() error")
	require.NotNil(t, gm.BookID)
	require.Equal(t, book.ID, *gm.BookID)
}

func TestGetPendingGoodreadsMetadataByBook(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	// Create a pending metadata record for this book
	title := "Remote Title"
	gm, err := d.CreateGoodreadsMetadata(t.Context(), user.ID,
		GoodreadsMetadataInput{BookID: &book.ID, Title: &title},
	)
	require.NoError(t, err)

	found, err := d.GetPendingGoodreadsMetadataByBook(t.Context(), user.ID, book.ID)
	require.NoError(t, err)
	require.Equal(t, gm.ID, found.ID)
	require.Equal(t, GoodreadsMetadataStatusPending, found.Status)
}

func TestGetPendingGoodreadsMetadataByBook_NotFound(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	_, err = d.GetPendingGoodreadsMetadataByBook(t.Context(), user.ID, "nonexistent")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetPendingGoodreadsMetadataByBook_IgnoresApplied(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	title := "Applied Title"
	gm, err := d.CreateGoodreadsMetadata(t.Context(), user.ID,
		GoodreadsMetadataInput{BookID: &book.ID, Title: &title},
	)
	require.NoError(t, err)

	_, err = d.UpdateGoodreadsMetadataStatus(t.Context(), user.ID, gm.ID, GoodreadsMetadataStatusApplied)
	require.NoError(t, err)

	_, err = d.GetPendingGoodreadsMetadataByBook(t.Context(), user.ID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetPendingGoodreadsMetadataByBook_MultiplePending(t *testing.T) {
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "hash")
	require.NoError(t, err)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	title1 := "First"
	gm1, err := d.CreateGoodreadsMetadata(t.Context(), user.ID,
		GoodreadsMetadataInput{BookID: &book.ID, Title: &title1},
	)
	require.NoError(t, err)

	title2 := "Second"
	gm2, err := d.CreateGoodreadsMetadata(t.Context(), user.ID,
		GoodreadsMetadataInput{BookID: &book.ID, Title: &title2},
	)
	require.NoError(t, err)

	// Should return one of the pending records (both are valid).
	found, err := d.GetPendingGoodreadsMetadataByBook(t.Context(), user.ID, book.ID)
	require.NoError(t, err)
	require.Equal(t, GoodreadsMetadataStatusPending, found.Status)
	require.Contains(t, []string{gm1.ID, gm2.ID}, found.ID)
}
