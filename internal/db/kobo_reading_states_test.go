package db

import (
	"testing"
	"time"
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
	b, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("createTestBookForKobo(%q): %v", title, err)
	}
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
	if err != nil {
		t.Fatalf("UpsertKoboReadingState() error: %v", err)
	}
	if state.ID == "" {
		t.Error("state.ID is empty")
	}
	if state.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", state.UserID, user.ID)
	}
	if state.BookID != book.ID {
		t.Errorf("BookID = %q, want %q", state.BookID, book.ID)
	}
	if state.Status != "Reading" {
		t.Errorf("Status = %q, want Reading", state.Status)
	}
	if state.PercentRead == nil || *state.PercentRead != 0.42 {
		t.Errorf("PercentRead = %v, want 0.42", state.PercentRead)
	}
	if state.LocationValue == nil || *state.LocationValue != "chapter-5" {
		t.Errorf("LocationValue = %v, want chapter-5", state.LocationValue)
	}
	if state.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestUpsertKoboReadingState_Updates(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Foundation")

	pct1 := 0.25
	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "Reading", &pct1, nil, nil, nil); err != nil {
		t.Fatalf("initial UpsertKoboReadingState() error: %v", err)
	}

	pct2 := 1.0
	updated, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "Finished", &pct2, nil, nil, nil)
	if err != nil {
		t.Fatalf("update UpsertKoboReadingState() error: %v", err)
	}
	if updated.Status != "Finished" {
		t.Errorf("Status = %q, want Finished", updated.Status)
	}
	if updated.PercentRead == nil || *updated.PercentRead != 1.0 {
		t.Errorf("PercentRead = %v, want 1.0", updated.PercentRead)
	}
}

func TestUpsertKoboReadingState_NilOptionalFields(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Naked Book")

	state, err := d.UpsertKoboReadingState(t.Context(), user.ID, book.ID, "ReadyToRead", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpsertKoboReadingState() error: %v", err)
	}
	if state.PercentRead != nil {
		t.Errorf("PercentRead = %v, want nil", state.PercentRead)
	}
	if state.LocationValue != nil {
		t.Errorf("LocationValue = %v, want nil", state.LocationValue)
	}
}

func TestGetKoboReadingState_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	book := createTestBookForKobo(t, d, "Unread Book")

	_, err := d.GetKoboReadingState(t.Context(), user.ID, book.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetKoboReadingState_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "User Two", "user2kobo@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser(user2): %v", err)
	}
	book := createTestBookForKobo(t, d, "Shared Book")

	if _, err := d.UpsertKoboReadingState(t.Context(), user1.ID, book.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(user1): %v", err)
	}

	// user2 should not see user1's state.
	_, err = d.GetKoboReadingState(t.Context(), user2.ID, book.ID)
	if err == nil {
		t.Error("GetKoboReadingState should return error for user2 who has no state")
	}
}

// ---- ListKoboReadingStatesSince ----

func TestListKoboReadingStatesSince_ZeroTimeReturnsAll(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Book One")
	b2 := createTestBookForKobo(t, d, "Book Two")
	b3 := createTestBookForKobo(t, d, "Book Three")

	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b1): %v", err)
	}
	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b2.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b2): %v", err)
	}
	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b3.ID, "ReadyToRead", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b3): %v", err)
	}

	states, err := d.ListKoboReadingStatesSince(t.Context(), user.ID, time.Time{})
	if err != nil {
		t.Fatalf("ListKoboReadingStatesSince(zero) error: %v", err)
	}
	if len(states) != 3 {
		t.Errorf("len(states) = %d, want 3", len(states))
	}
}

func TestListKoboReadingStatesSince_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Iso User", "iso@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser(user2): %v", err)
	}
	book := createTestBookForKobo(t, d, "Isolation Book")

	if _, err := d.UpsertKoboReadingState(t.Context(), user1.ID, book.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(user1): %v", err)
	}

	states, err := d.ListKoboReadingStatesSince(t.Context(), user2.ID, time.Time{})
	if err != nil {
		t.Fatalf("ListKoboReadingStatesSince(user2) error: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("user2 should see 0 states, got %d", len(states))
	}
}

func TestListKoboReadingStatesSince_FiltersByTime(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Early Book")
	b2 := createTestBookForKobo(t, d, "Late Book")

	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b1): %v", err)
	}
	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b2.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b2): %v", err)
	}

	// A since value in the future should exclude all states.
	futureTime := koboTestFutureTime()
	states, err := d.ListKoboReadingStatesSince(t.Context(), user.ID, futureTime)
	if err != nil {
		t.Fatalf("ListKoboReadingStatesSince(future) error: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("len(states) = %d, want 0 (all before future cutoff)", len(states))
	}

	// A since value in the distant past should include all states.
	pastTime := koboTestPastTime()
	statesAll, err := d.ListKoboReadingStatesSince(t.Context(), user.ID, pastTime)
	if err != nil {
		t.Fatalf("ListKoboReadingStatesSince(past) error: %v", err)
	}
	if len(statesAll) != 2 {
		t.Errorf("len(statesAll) = %d, want 2 (all after past cutoff)", len(statesAll))
	}
}

// ---- GetReadingStatesForBooks ----

func TestGetReadingStatesForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{}, time.Time{})
	if err != nil {
		t.Fatalf("GetReadingStatesForBooks(empty) error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil for empty input", result)
	}
}

func TestGetReadingStatesForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, nil, time.Time{})
	if err != nil {
		t.Fatalf("GetReadingStatesForBooks(nil) error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil for nil input", result)
	}
}

func TestGetReadingStatesForBooks_ReturnsMap(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Map Book A")
	b2 := createTestBookForKobo(t, d, "Map Book B")
	b3 := createTestBookForKobo(t, d, "Map Book C")

	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b1): %v", err)
	}
	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b3.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b3): %v", err)
	}
	// b2 has no state.

	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{b1.ID, b2.ID, b3.ID}, time.Time{})
	if err != nil {
		t.Fatalf("GetReadingStatesForBooks() error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}
	if result[b1.ID] == nil {
		t.Errorf("result[b1] = nil, want state")
	}
	if result[b1.ID] != nil && result[b1.ID].Status != "Reading" {
		t.Errorf("b1 Status = %q, want Reading", result[b1.ID].Status)
	}
	if result[b3.ID] == nil {
		t.Errorf("result[b3] = nil, want state")
	}
	if result[b2.ID] != nil {
		t.Errorf("result[b2] should be absent, got %v", result[b2.ID])
	}
}

func TestGetReadingStatesForBooks_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Other Kobo User", "other_kobo@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser(user2): %v", err)
	}
	book := createTestBookForKobo(t, d, "Isolated Book")

	if _, err := d.UpsertKoboReadingState(t.Context(), user1.ID, book.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(user1): %v", err)
	}

	result, err := d.GetReadingStatesForBooks(t.Context(), user2.ID, []string{book.ID}, time.Time{})
	if err != nil {
		t.Fatalf("GetReadingStatesForBooks(user2) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("user2 should see 0 results, got %d", len(result))
	}
}

func TestGetReadingStatesForBooks_WithSinceFilter(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)
	b1 := createTestBookForKobo(t, d, "Since Book A")
	b2 := createTestBookForKobo(t, d, "Since Book B")

	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, "Reading", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b1): %v", err)
	}
	if _, err := d.UpsertKoboReadingState(t.Context(), user.ID, b2.ID, "Finished", nil, nil, nil, nil); err != nil {
		t.Fatalf("UpsertKoboReadingState(b2): %v", err)
	}

	// A since value in the future should exclude all states.
	result, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{b1.ID, b2.ID}, koboTestFutureTime())
	if err != nil {
		t.Fatalf("GetReadingStatesForBooks(future since) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 (all before future cutoff)", len(result))
	}

	// A since value in the distant past should include all states.
	resultAll, err := d.GetReadingStatesForBooks(t.Context(), user.ID, []string{b1.ID, b2.ID}, koboTestPastTime())
	if err != nil {
		t.Fatalf("GetReadingStatesForBooks(past since) error: %v", err)
	}
	if len(resultAll) != 2 {
		t.Errorf("len(resultAll) = %d, want 2 (all after past cutoff)", len(resultAll))
	}
}
