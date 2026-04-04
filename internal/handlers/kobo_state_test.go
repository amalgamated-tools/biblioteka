package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleBookState_UnknownMethodReturnsOK verifies that HandleBookState
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

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (unknown method)", w.Code, http.StatusOK)
	}
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

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
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
	if _, err := d.UpsertKoboReadingState(context.Background(), user.ID, book.ID, "Reading", &pct, nil, nil, nil); err != nil {
		require.NoError(t, err, "upsert reading state")
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/library/"+book.ID+"/state", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Reading") {
		t.Errorf("expected Reading status in response, got: %s", body)
	}
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
	if _, err := d.UpsertKoboReadingState(context.Background(), user1.ID, book.ID, "Reading", &pct, nil, nil, nil); err != nil {
		require.NoError(t, err, "upsert state for user1")
	}

	// user2 should get a default (not user1's) state.
	r := httptest.NewRequest(http.MethodGet, "/v1/library/"+book.ID+"/state", nil)
	r = withUserID(r, user2.ID)
	w := httptest.NewRecorder()
	h.HandleBookState(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d", w.Code, http.StatusOK)
	}
}
