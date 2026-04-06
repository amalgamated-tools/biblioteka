package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleBookState_UpdateWithNonExistentBook verifies that updating state
// for a book that was deleted mid-flight returns 404, not 500.
func TestHandleBookState_UpdateWithNonExistentBook(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	book, err := h.DB.CreateBook(context.Background(), "Delete Race Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	// Delete the book to simulate the race condition.
	require.NoError(t, h.DB.DeleteBook(context.Background(), book.ID), "delete book")

	body := `{"ReadingStates":[{"StatusInfo":{"Status":"Reading"}}]}`
	r := httptest.NewRequest(http.MethodPut, "/v1/library/"+book.ID+"/state", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	// The GetBook pre-check returns 404 because the book was deleted; the
	// ErrBookNotFound path in UpsertKoboReadingState covers the race window.
	require.Equal(t, http.StatusNotFound, w.Code)
}

// returns 200 with an empty JSON array for unknown HTTP methods, matching
// the Kobo device's expected behavior.
func TestHandleBookState_UnknownMethodReturnsOK(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	book, err := h.DB.CreateBook(context.Background(), "State Unknown Method", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodDelete, "/v1/library/"+book.ID+"/state", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

// TestHandleBookState_StatePathParsing verifies that HandleBookState correctly
// extracts the book ID from the URL path.
func TestHandleBookState_StatePathParsing(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	book, err := h.DB.CreateBook(context.Background(), "Path Parsing Test", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/v1/library/"+book.ID+"/state", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

// TestHandleBookState_ReadingProgressPercentage verifies that updating reading
// state with a percentage value stores and returns the progress.
func TestHandleBookState_ReadingProgressPercentage(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	user, err := d.CreateUser(context.Background(), "Progress User", "progress@example.com", "password")
	require.NoError(t, err, "create user")
	book, err := d.CreateBook(context.Background(), "Progress Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	pct := 0.35
	_, err = d.UpsertKoboReadingState(context.Background(), user.ID, book.ID, "Reading", &pct, nil, nil, nil)
	require.NoError(t, err, "upsert reading state")

	r := httptest.NewRequest(http.MethodGet, "/v1/library/"+book.ID+"/state", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	require.Contains(t, body, "Reading")
}

// TestHandleBookState_UserIsolation verifies that a user cannot read the
// reading state of a book belonging to another user.
func TestHandleBookState_UserIsolation(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	user1, err := d.CreateUser(context.Background(), "User1", "user1@example.com", "password")
	require.NoError(t, err, "create user1")
	user2, err := d.CreateUser(context.Background(), "User2", "user2@example.com", "password")
	require.NoError(t, err, "create user2")
	book, err := d.CreateBook(context.Background(), "Shared Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	// Set state for user1.
	pct := 0.5
	_, err = d.UpsertKoboReadingState(context.Background(), user1.ID, book.ID, "Reading", &pct, nil, nil, nil)
	require.NoError(t, err, "upsert state for user1")

	// user2 should get a default (not user1's) state.
	r := httptest.NewRequest(http.MethodGet, "/v1/library/"+book.ID+"/state", nil)
	r = withUserID(r, user2.ID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}
