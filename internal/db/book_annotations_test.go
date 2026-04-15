package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestUserForAnnotation(t *testing.T, d *DB, email string) string {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", email, "password123")
	require.NoError(t, err, "createTestUserForAnnotation")
	return u.ID
}

func createTestBookForAnnotation(t *testing.T, d *DB, title string) string {
	t.Helper()
	b, err := d.CreateBook(t.Context(), BookInput{Title: title})
	require.NoError(t, err, "createTestBookForAnnotation")
	return b.ID
}

func TestCreateAnnotation(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForAnnotation(t, d, "user@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	a, err := d.CreateAnnotation(t.Context(), userID, bookID, "Great chapter!", nil, nil)
	require.NoError(t, err)
	require.NotEmpty(t, a.ID)
	require.Equal(t, userID, a.UserID)
	require.Equal(t, bookID, a.BookID)
	require.Equal(t, "Great chapter!", a.Text)
	require.Nil(t, a.GroupID)
}

func TestCreateAnnotation_WithGroup(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForAnnotation(t, d, "user@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	g, err := d.CreateGroup(t.Context(), userID, "Book Club", nil)
	require.NoError(t, err)

	a, err := d.CreateAnnotation(t.Context(), userID, bookID, "Shared note", nil, &g.ID)
	require.NoError(t, err)
	require.NotNil(t, a.GroupID)
	require.Equal(t, g.ID, *a.GroupID)
}

func TestCreateAnnotation_NonMemberCannotShareToGroup(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForAnnotation(t, d, "owner@example.com")
	otherID := createTestUserForAnnotation(t, d, "other@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	// otherID is not a member of the group.
	_, err = d.CreateAnnotation(t.Context(), otherID, bookID, "Injected note", nil, &g.ID)
	require.ErrorIs(t, err, ErrNotGroupMember)
}

func TestUpdateAnnotation_NonMemberCannotShareToGroup(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForAnnotation(t, d, "owner@example.com")
	otherID := createTestUserForAnnotation(t, d, "other@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	// Create a private annotation first.
	a, err := d.CreateAnnotation(t.Context(), otherID, bookID, "My note", nil, nil)
	require.NoError(t, err)

	// Try to update it to share with a group the user is not a member of.
	_, err = d.UpdateAnnotation(t.Context(), a.ID, otherID, "Updated", nil, &g.ID)
	require.ErrorIs(t, err, ErrNotGroupMember)
}

func TestListAnnotationsForBook_UserSeesOwnAnnotations(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForAnnotation(t, d, "user@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	_, err := d.CreateAnnotation(t.Context(), userID, bookID, "Note 1", nil, nil)
	require.NoError(t, err)
	_, err = d.CreateAnnotation(t.Context(), userID, bookID, "Note 2", nil, nil)
	require.NoError(t, err)

	annotations, err := d.ListAnnotationsForBook(t.Context(), bookID, userID)
	require.NoError(t, err)
	require.Len(t, annotations, 2)
}

func TestListAnnotationsForBook_MemberSeesGroupAnnotations(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForAnnotation(t, d, "owner@example.com")
	memberID := createTestUserForAnnotation(t, d, "member@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	_, err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	// Owner creates a group-shared annotation.
	_, err = d.CreateAnnotation(t.Context(), ownerID, bookID, "Group note", nil, &g.ID)
	require.NoError(t, err)

	// Member should see the group annotation.
	annotations, err := d.ListAnnotationsForBook(t.Context(), bookID, memberID)
	require.NoError(t, err)
	require.Len(t, annotations, 1)
	require.Equal(t, "Group note", annotations[0].Text)
}

func TestListAnnotationsForBook_NonMemberCannotSeeGroupAnnotations(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForAnnotation(t, d, "owner@example.com")
	nonMemberID := createTestUserForAnnotation(t, d, "nonmember@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	// Owner creates a group-shared annotation.
	_, err = d.CreateAnnotation(t.Context(), ownerID, bookID, "Group note", nil, &g.ID)
	require.NoError(t, err)

	// Non-member should NOT see the group annotation.
	annotations, err := d.ListAnnotationsForBook(t.Context(), bookID, nonMemberID)
	require.NoError(t, err)
	require.Empty(t, annotations)
}

func TestDeleteAnnotation(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForAnnotation(t, d, "user@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	a, err := d.CreateAnnotation(t.Context(), userID, bookID, "Note", nil, nil)
	require.NoError(t, err)

	err = d.DeleteAnnotation(t.Context(), a.ID, userID)
	require.NoError(t, err)

	annotations, err := d.ListAnnotationsForBook(t.Context(), bookID, userID)
	require.NoError(t, err)
	require.Empty(t, annotations)
}

func TestDeleteAnnotation_OtherUserCannotDelete(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForAnnotation(t, d, "user@example.com")
	otherID := createTestUserForAnnotation(t, d, "other@example.com")
	bookID := createTestBookForAnnotation(t, d, "Test Book")

	a, err := d.CreateAnnotation(t.Context(), userID, bookID, "Note", nil, nil)
	require.NoError(t, err)

	err = d.DeleteAnnotation(t.Context(), a.ID, otherID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
