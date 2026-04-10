package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// koboTestFutureTime returns a time well in the future that should be after
// any timestamp written by tests in this session.
func koboTestFutureTime() time.Time {
	return time.Now().UTC().Add(24 * time.Hour)
}

// koboTestPastTime returns a time well in the past that should be before
// any timestamp written by tests in this session.
func koboTestPastTime() time.Time {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
}

// ---- helpers ----

func createTestBookForKobo(t *testing.T, d *DB, title string) *Book {
	t.Helper()
	b, err := d.CreateBook(t.Context(), BookInput{Title: title})
	require.NoError(t, err, "createTestBookForKobo(%q)", title)
	return b
}

// ---- UpsertKoboReadingState / GetKoboReadingState ----

func TestUpsertKoboReadingState_Creates(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Dune")

	pct := 0.42
	locVal := "chapter-5"
	locType := "chapter"
	locSrc := "toc"

	state, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "Reading", &pct, &locVal, &locType, &locSrc)
	require.NoError(t, err, "UpsertKoboReadingState() error")
	require.NotEqual(t, "", state.ID)
	require.Equal(t, user.ID, state.UserID)
	require.Equal(t, book.ID, state.BookID)
	require.Equal(t, "Reading", state.Status)
	require.NotNil(t, state.PercentRead)
	require.Equal(t, 0.42, *state.PercentRead)
	require.NotNil(t, state.LocationValue)
	require.Equal(t, "chapter-5", *state.LocationValue)
	require.False(t, state.CreatedAt.IsZero())
}

func TestUpsertKoboReadingState_Updates(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Foundation")

	pct1 := 0.25
	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "Reading", &pct1, nil, nil, nil)
	require.NoError(t, err, "initial UpsertKoboReadingState() error")

	pct2 := 1.0
	updated, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "Finished", &pct2, nil, nil, nil)
	require.NoError(t, err, "update UpsertKoboReadingState() error")
	require.Equal(t, "Finished", updated.Status)
	require.NotNil(t, updated.PercentRead)
	require.Equal(t, 1.0, *updated.PercentRead)
}

func TestUpsertKoboReadingState_NilOptionalFields(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Naked Book")

	state, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "ReadyToRead", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState() error")
	require.Nil(t, state.PercentRead)
	require.Nil(t, state.LocationValue)
}

func TestGetKoboReadingState_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Unread Book")

	_, err := d.GetKoboReadingState(t.Context(), user.ID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetKoboReadingState_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "User Two", "user2kobo@example.com", "pass")
	require.NoError(t, err, "CreateUser(user2)")
	book := createTestBookForKobo(t, d, "Shared Book")

	_, err = d.UpsertKoboReadingState(t.Context(), user1.ID, book.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(user1)")

	// user2 should not see user1's state.
	_, err = d.GetKoboReadingState(t.Context(), user2.ID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// ---- ListKoboReadingStatesSince ----

func TestListKoboReadingStatesSince_ZeroTimeReturnsAll(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Book One")
	b2 := createTestBookForKobo(t, d, "Book Two")
	b3 := createTestBookForKobo(t, d, "Book Three")

	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b1)")
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b2.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b2)")
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b3.ID, "ReadyToRead", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b3)")

	states, err := d.ListKoboReadingStatesSince(t.Context(), user.ID, time.Time{})
	require.NoError(t, err, "ListKoboReadingStatesSince(zero) error")
	require.Len(t, states, 3)
}

func TestListKoboReadingStatesSince_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Iso User", "iso@example.com", "pass")
	require.NoError(t, err, "CreateUser(user2)")
	book := createTestBookForKobo(t, d, "Isolation Book")

	_, err = d.UpsertKoboReadingState(t.Context(), user1.ID, book.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(user1)")

	states, err := d.ListKoboReadingStatesSince(t.Context(), user2.ID, time.Time{})
	require.NoError(t, err, "ListKoboReadingStatesSince(user2) error")
	require.Len(t, states, 0)
}

func TestListKoboReadingStatesSince_FiltersByTime(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Early Book")
	b2 := createTestBookForKobo(t, d, "Late Book")

	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b1)")
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b2.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b2)")

	// A since value in the future should exclude all states.
	futureTime := koboTestFutureTime()
	states, err := d.ListKoboReadingStatesSince(t.Context(), user.ID, futureTime)
	require.NoError(t, err, "ListKoboReadingStatesSince(future) error")
	require.Len(t, states, 0)

	// A since value in the distant past should include all states.
	pastTime := koboTestPastTime()
	statesAll, err := d.ListKoboReadingStatesSince(t.Context(), user.ID, pastTime)
	require.NoError(t, err, "ListKoboReadingStatesSince(past) error")
	require.Len(t, statesAll, 2)
}

// ---- GetReadingStatesForBooks ----

func TestGetReadingStatesForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{}, time.Time{})
	require.NoError(t, err, "GetReadingStatesForBooks(empty) error")
	require.Nil(t, result)
}

func TestGetReadingStatesForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, nil, time.Time{})
	require.NoError(t, err, "GetReadingStatesForBooks(nil) error")
	require.Nil(t, result)
}

func TestGetReadingStatesForBooks_ReturnsMap(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Map Book A")
	b2 := createTestBookForKobo(t, d, "Map Book B")
	b3 := createTestBookForKobo(t, d, "Map Book C")

	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b1)")
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b3.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b3)")
	// b2 has no state.

	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{b1.ID, b2.ID, b3.ID}, time.Time{})
	require.NoError(t, err, "GetReadingStatesForBooks() error")
	require.Len(t, result, 2)
	require.NotNil(t, result[b1.ID])
	require.True(t, result[b1.ID] != nil)
	require.Equal(t, "Reading", result[b1.ID].Status)
	require.NotNil(t, result[b3.ID])
	require.Nil(t, result[b2.ID])
}

func TestGetReadingStatesForBooks_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Other Kobo User", "other_kobo@example.com", "pass")
	require.NoError(t, err, "CreateUser(user2)")
	book := createTestBookForKobo(t, d, "Isolated Book")

	_, err = d.UpsertKoboReadingState(t.Context(), user1.ID, book.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(user1)")

	result, err := d.GetReadingStatesForBooks(t.Context(), user2.ID, []string{book.ID}, time.Time{})
	require.NoError(t, err, "GetReadingStatesForBooks(user2) error")
	require.Len(t, result, 0)
}

func TestUpsertKoboReadingState_NonExistentBookReturnsErrBookNotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, "nonexistent-book-id", "Reading", nil, nil, nil, nil)
	require.ErrorIs(t, err, ErrBookNotFound)
}

func TestGetReadingStatesForBooks_WithSinceFilter(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Since Book A")
	b2 := createTestBookForKobo(t, d, "Since Book B")

	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b1)")
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b2.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err, "UpsertKoboReadingState(b2)")

	// A since value in the future should exclude all states.
	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{b1.ID, b2.ID}, koboTestFutureTime())
	require.NoError(t, err, "GetReadingStatesForBooks(future since) error")
	require.Len(t, result, 0)

	// A since value in the distant past should include all states.
	resultAll, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{b1.ID, b2.ID}, koboTestPastTime())
	require.NoError(t, err, "GetReadingStatesForBooks(past since) error")
	require.Len(t, resultAll, 2)
}
