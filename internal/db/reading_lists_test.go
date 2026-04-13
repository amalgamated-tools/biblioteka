package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestUserForRL is a helper that creates a test user and returns its ID.
func createTestUserForRL(t *testing.T, d *DB, email string) string {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", email, "password123")
	require.NoError(t, err, "createTestUserForRL")
	return u.ID
}

func TestCreateReadingList(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "To Read", nil)
	require.NoError(t, err, "CreateReadingList() error")
	require.NotEmpty(t, rl.ID)
	require.Equal(t, userID, rl.UserID)
	require.Equal(t, "To Read", rl.Name)
	require.Nil(t, rl.Description)
	require.Equal(t, 0, rl.BookCount)
	require.False(t, rl.CreatedAt.IsZero())
}

func TestCreateReadingList_WithDescription(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	desc := "Books I plan to read this year"
	rl, err := d.CreateReadingList(t.Context(), userID, "To Read", &desc)
	require.NoError(t, err)
	require.NotNil(t, rl.Description)
	require.Equal(t, desc, *rl.Description)
}

func TestCreateReadingList_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "  To   Read  ", nil)
	require.NoError(t, err)
	require.Equal(t, "To Read", rl.Name)
}

func TestCreateReadingList_BlankName(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.CreateReadingList(t.Context(), userID, name, nil)
		require.ErrorIs(t, err, ErrInvalidReadingListName, "name=%q", name)
	}
}

func TestCreateReadingList_DuplicateNameSameUser(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	_, err := d.CreateReadingList(t.Context(), userID, "Favorites", nil)
	require.NoError(t, err)

	_, err = d.CreateReadingList(t.Context(), userID, "Favorites", nil)
	require.ErrorIs(t, err, ErrReadingListNameExists)
}

func TestCreateReadingList_SameNameDifferentUsers(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	// Two users can have reading lists with the same name.
	_, err := d.CreateReadingList(t.Context(), user1ID, "Favorites", nil)
	require.NoError(t, err)
	_, err = d.CreateReadingList(t.Context(), user2ID, "Favorites", nil)
	require.NoError(t, err)
}

func TestGetReadingList(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	created, err := d.CreateReadingList(t.Context(), userID, "To Read", nil)
	require.NoError(t, err)

	got, err := d.GetReadingList(t.Context(), created.ID, userID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "To Read", got.Name)
}

func TestGetReadingList_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	_, err := d.GetReadingList(t.Context(), "nonexistent-id", userID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetReadingList_OtherUserCannotAccess(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl, err := d.CreateReadingList(t.Context(), user1ID, "My List", nil)
	require.NoError(t, err)

	_, err = d.GetReadingList(t.Context(), rl.ID, user2ID)
	require.ErrorIs(t, err, sql.ErrNoRows, "other user should not be able to access the list")
}

func TestListReadingLists(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	_, err := d.CreateReadingList(t.Context(), userID, "Zebra", nil)
	require.NoError(t, err)
	_, err = d.CreateReadingList(t.Context(), userID, "Alpha", nil)
	require.NoError(t, err)

	lists, err := d.ListReadingLists(t.Context(), userID)
	require.NoError(t, err)
	require.Len(t, lists, 2)
	require.Equal(t, "Alpha", lists[0].Name, "lists should be ordered by name")
	require.Equal(t, "Zebra", lists[1].Name)
}

func TestListReadingLists_IsolatesUsers(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	_, err := d.CreateReadingList(t.Context(), user1ID, "User1 List", nil)
	require.NoError(t, err)

	lists, err := d.ListReadingLists(t.Context(), user2ID)
	require.NoError(t, err)
	require.Empty(t, lists, "user2 should see no lists")
}

func TestUpdateReadingList(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "Old Name", nil)
	require.NoError(t, err)

	desc := "Updated desc"
	updated, err := d.UpdateReadingList(t.Context(), rl.ID, userID, "New Name", &desc)
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.Name)
	require.NotNil(t, updated.Description)
	require.Equal(t, "Updated desc", *updated.Description)
}

func TestUpdateReadingList_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	_, err := d.UpdateReadingList(t.Context(), "nonexistent", userID, "Name", nil)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpdateReadingList_BlankName(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "Original", nil)
	require.NoError(t, err)

	_, err = d.UpdateReadingList(t.Context(), rl.ID, userID, "  ", nil)
	require.ErrorIs(t, err, ErrInvalidReadingListName)
}

func TestUpdateReadingList_DuplicateName(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	_, err := d.CreateReadingList(t.Context(), userID, "First", nil)
	require.NoError(t, err)
	second, err := d.CreateReadingList(t.Context(), userID, "Second", nil)
	require.NoError(t, err)

	_, err = d.UpdateReadingList(t.Context(), second.ID, userID, "First", nil)
	require.ErrorIs(t, err, ErrReadingListNameExists)
}

func TestUpdateReadingList_OtherUserCannotUpdate(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl, err := d.CreateReadingList(t.Context(), user1ID, "My List", nil)
	require.NoError(t, err)

	_, err = d.UpdateReadingList(t.Context(), rl.ID, user2ID, "Hacked", nil)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteReadingList(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "To Delete", nil)
	require.NoError(t, err)

	err = d.DeleteReadingList(t.Context(), rl.ID, userID)
	require.NoError(t, err)

	_, err = d.GetReadingList(t.Context(), rl.ID, userID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteReadingList_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	err := d.DeleteReadingList(t.Context(), "nonexistent", userID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteReadingList_OtherUserCannotDelete(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl, err := d.CreateReadingList(t.Context(), user1ID, "My List", nil)
	require.NoError(t, err)

	err = d.DeleteReadingList(t.Context(), rl.ID, user2ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestAddBookToReadingList(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	added, err := d.AddBookToReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)
	require.True(t, added)

	got, err := d.GetReadingList(t.Context(), rl.ID, userID)
	require.NoError(t, err)
	require.Equal(t, 1, got.BookCount)
}

func TestAddBookToReadingList_Idempotent(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	added, err := d.AddBookToReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)
	require.True(t, added)
	// Adding the same book again is idempotent.
	added, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)
	require.False(t, added, "duplicate add should return false")

	got, err := d.GetReadingList(t.Context(), rl.ID, userID)
	require.NoError(t, err)
	require.Equal(t, 1, got.BookCount, "duplicate add should not increase count")
}

func TestAddBookToReadingList_ListNotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	_, err = d.AddBookToReadingList(t.Context(), "nonexistent", userID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestAddBookToReadingList_BookNotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)

	_, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, "nonexistent")
	require.ErrorIs(t, err, ErrBookNotFound)
}

func TestAddBookToReadingList_OtherUserCannotAdd(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl, err := d.CreateReadingList(t.Context(), user1ID, "My List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	_, err = d.AddBookToReadingList(t.Context(), rl.ID, user2ID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestRemoveBookFromReadingList(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	_, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)
	removed, err := d.RemoveBookFromReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)
	require.True(t, removed)

	got, err := d.GetReadingList(t.Context(), rl.ID, userID)
	require.NoError(t, err)
	require.Equal(t, 0, got.BookCount)
}

func TestRemoveBookFromReadingList_NotPresent(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	removed, err := d.RemoveBookFromReadingList(t.Context(), rl.ID, userID, book.ID)
	require.NoError(t, err)
	require.False(t, removed, "removing a book not in the list should return false")
}

func TestRemoveBookFromReadingList_OtherUserCannotRemove(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl, err := d.CreateReadingList(t.Context(), user1ID, "My List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)
	_, err = d.AddBookToReadingList(t.Context(), rl.ID, user1ID, book.ID)
	require.NoError(t, err)

	_, err = d.RemoveBookFromReadingList(t.Context(), rl.ID, user2ID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListReadingListBooks(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book1, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)
	book2, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)

	_, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, book1.ID)
	require.NoError(t, err)
	_, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, book2.ID)
	require.NoError(t, err)

	books, total, err := d.ListReadingListBooks(t.Context(), rl.ID, userID, 50, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, books, 2)
}

func TestListReadingListBooks_ListNotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	_, _, err := d.ListReadingListBooks(t.Context(), "nonexistent", userID, 50, 0)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListReadingListBooks_OtherUserSeesForbidden(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl, err := d.CreateReadingList(t.Context(), user1ID, "My List", nil)
	require.NoError(t, err)

	_, _, err = d.ListReadingListBooks(t.Context(), rl.ID, user2ID, 50, 0)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetReadingListsForBook(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl1, err := d.CreateReadingList(t.Context(), userID, "Alpha", nil)
	require.NoError(t, err)
	rl2, err := d.CreateReadingList(t.Context(), userID, "Beta", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	_, err = d.AddBookToReadingList(t.Context(), rl1.ID, userID, book.ID)
	require.NoError(t, err)
	_, err = d.AddBookToReadingList(t.Context(), rl2.ID, userID, book.ID)
	require.NoError(t, err)

	lists, err := d.GetReadingListsForBook(t.Context(), book.ID, userID)
	require.NoError(t, err)
	require.Len(t, lists, 2)
	require.Equal(t, "Alpha", lists[0].Name)
	require.Equal(t, "Beta", lists[1].Name)
}

func TestGetReadingListsForBook_IsolatesUsers(t *testing.T) {
	d := newTestDB(t)
	user1ID := createTestUserForRL(t, d, "user1@example.com")
	user2ID := createTestUserForRL(t, d, "user2@example.com")

	rl1, err := d.CreateReadingList(t.Context(), user1ID, "User1 List", nil)
	require.NoError(t, err)
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)
	_, err = d.AddBookToReadingList(t.Context(), rl1.ID, user1ID, book.ID)
	require.NoError(t, err)

	lists, err := d.GetReadingListsForBook(t.Context(), book.ID, user2ID)
	require.NoError(t, err)
	require.Empty(t, lists, "user2 should see no lists")
}

func TestListReadingLists_IncludesBookCount(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForRL(t, d, "user@example.com")

	rl, err := d.CreateReadingList(t.Context(), userID, "My List", nil)
	require.NoError(t, err)
	book1, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)
	book2, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)
	_, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, book1.ID)
	require.NoError(t, err)
	_, err = d.AddBookToReadingList(t.Context(), rl.ID, userID, book2.ID)
	require.NoError(t, err)

	lists, err := d.ListReadingLists(t.Context(), userID)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	require.Equal(t, 2, lists[0].BookCount)
}
