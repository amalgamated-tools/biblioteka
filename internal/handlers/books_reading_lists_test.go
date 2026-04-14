package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// TestGetBookReadingLists_Empty verifies that the endpoint returns an empty
// slice (not null) when the book has not been added to any reading list.
func TestGetBookReadingLists_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/reading-lists", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.NotNil(t, dtos, "response must not be null")
	require.Empty(t, dtos)
}

// TestGetBookReadingLists_WithLists verifies that the endpoint returns the
// reading lists that contain the given book.
func TestGetBookReadingLists_WithLists(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err, "create book")

	list1, err := h.DB.CreateReadingList(t.Context(), userID, "Science Fiction Classics", nil)
	require.NoError(t, err, "create list1")
	list2, err := h.DB.CreateReadingList(t.Context(), userID, "All-Time Favorites", nil)
	require.NoError(t, err, "create list2")
	list3, err := h.DB.CreateReadingList(t.Context(), userID, "Wishlist", nil)
	require.NoError(t, err, "create list3")

	_, err = h.DB.AddBookToReadingList(t.Context(), list1.ID, userID, b.ID)
	require.NoError(t, err, "add book to list1")
	_, err = h.DB.AddBookToReadingList(t.Context(), list2.ID, userID, b.ID)
	require.NoError(t, err, "add book to list2")
	// list3 intentionally does not contain the book.
	_ = list3

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/reading-lists", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 2, "only the two lists containing the book should be returned")

	names := make([]string, len(dtos))
	for i, d := range dtos {
		names[i] = d.Name
	}
	require.ElementsMatch(t, []string{"Science Fiction Classics", "All-Time Favorites"}, names)
}

// TestGetBookReadingLists_IsolatesUsers verifies that a user cannot see
// another user's reading lists for the same book.
func TestGetBookReadingLists_IsolatesUsers(t *testing.T) {
	h, user1ID := setupBookHandler(t)

	user2, err := h.DB.CreateUser(t.Context(), "Other User", "other@example.com", "password1")
	require.NoError(t, err, "create user2")

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Foundation"})
	require.NoError(t, err, "create book")

	// user1 adds the book to their list.
	list1, err := h.DB.CreateReadingList(t.Context(), user1ID, "User One List", nil)
	require.NoError(t, err, "create list for user1")
	_, err = h.DB.AddBookToReadingList(t.Context(), list1.ID, user1ID, b.ID)
	require.NoError(t, err, "add book to user1 list")

	// user2 also adds the book to their own list.
	list2, err := h.DB.CreateReadingList(t.Context(), user2.ID, "User Two List", nil)
	require.NoError(t, err, "create list for user2")
	_, err = h.DB.AddBookToReadingList(t.Context(), list2.ID, user2.ID, b.ID)
	require.NoError(t, err, "add book to user2 list")

	// Request as user2: must only see user2's list.
	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/reading-lists", nil)
	r = withUserID(r, user2.ID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dtos []readingListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 1, "only the authenticated user's lists should be returned")
	require.Equal(t, "User Two List", dtos[0].Name)
}

// TestGetBookReadingLists_MethodNotAllowed verifies that non-GET methods on the
// reading-lists sub-resource return 405.
func TestGetBookReadingLists_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Brave New World"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/reading-lists", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
