package db

import (
	"testing"
)

// These tests cover edge cases in book_relations.go that are not already
// exercised by books_test.go (which covers the basic happy-path scenarios).

// ---- GetBookAuthors ----

func TestGetBookAuthors_Empty(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Authorless Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	if err != nil {
		t.Fatalf("GetBookAuthors() error: %v", err)
	}
	if len(authors) != 0 {
		t.Errorf("len(authors) = %d, want 0", len(authors))
	}
}

// ---- SetBookAuthors ----

func TestSetBookAuthors_ClearAll(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	author, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateAuthor(): %v", err)
	}

	if err := d.SetBookAuthors(t.Context(), book.ID, []string{author.ID}); err != nil {
		t.Fatalf("SetBookAuthors(set): %v", err)
	}

	// Clear by passing an empty slice.
	if err := d.SetBookAuthors(t.Context(), book.ID, []string{}); err != nil {
		t.Fatalf("SetBookAuthors(clear): %v", err)
	}

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	if err != nil {
		t.Fatalf("GetBookAuthors() error: %v", err)
	}
	if len(authors) != 0 {
		t.Errorf("len(authors) = %d, want 0 after clear", len(authors))
	}
}

func TestSetBookAuthors_DeduplicatesIDs(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Shared Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	author, err := d.CreateAuthor(t.Context(), "Author A", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateAuthor(): %v", err)
	}

	// Pass the same author ID three times.
	if err := d.SetBookAuthors(t.Context(), book.ID, []string{author.ID, author.ID, author.ID}); err != nil {
		t.Fatalf("SetBookAuthors() with duplicates error: %v", err)
	}

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	if err != nil {
		t.Fatalf("GetBookAuthors() error: %v", err)
	}
	if len(authors) != 1 {
		t.Errorf("len(authors) = %d, want 1 after deduplication", len(authors))
	}
}

// ---- GetBookSeries ----

func TestGetBookSeries_Empty(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Standalone Novel", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}

	entries, err := d.GetBookSeries(t.Context(), book.ID)
	if err != nil {
		t.Fatalf("GetBookSeries() error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

// ---- SetBookSeries ----

func TestSetBookSeries_ClearAll(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	s, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}

	if err := d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{{SeriesID: s.ID}}); err != nil {
		t.Fatalf("SetBookSeries(set): %v", err)
	}

	// Clear by passing empty slice.
	if err := d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{}); err != nil {
		t.Fatalf("SetBookSeries(clear): %v", err)
	}

	entries, err := d.GetBookSeries(t.Context(), book.ID)
	if err != nil {
		t.Fatalf("GetBookSeries() error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0 after clear", len(entries))
	}
}

func TestSetBookSeries_DeduplicatesLastPositionWins(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Crossover Novel", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	s, err := d.CreateSeries(t.Context(), "A Series", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}

	// Pass the same series ID twice with different positions; last position wins.
	entries := []BookSeriesInput{
		{SeriesID: s.ID, Position: new(1.0)},
		{SeriesID: s.ID, Position: new(99.0)},
	}
	if err := d.SetBookSeries(t.Context(), book.ID, entries); err != nil {
		t.Fatalf("SetBookSeries() with duplicates error: %v", err)
	}

	got, err := d.GetBookSeries(t.Context(), book.ID)
	if err != nil {
		t.Fatalf("GetBookSeries() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1 after deduplication", len(got))
	}
	// The implementation processes entries in reverse order so the last
	// element (position 99) is the one that survives deduplication.
	if got[0].Position == nil || *got[0].Position != 99.0 {
		t.Errorf("position = %v, want 99.0 (last position wins)", got[0].Position)
	}
}

// ---- GetAuthorsForBooks ----

func TestGetAuthorsForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetAuthorsForBooks(t.Context(), []string{})
	if err != nil {
		t.Fatalf("GetAuthorsForBooks(empty) error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil for empty input", result)
	}
}

func TestGetAuthorsForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetAuthorsForBooks(t.Context(), nil)
	if err != nil {
		t.Fatalf("GetAuthorsForBooks(nil) error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil for nil input", result)
	}
}

func TestGetAuthorsForBooks_BookWithNoAuthors(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Orphan Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}

	result, err := d.GetAuthorsForBooks(t.Context(), []string{book.ID})
	if err != nil {
		t.Fatalf("GetAuthorsForBooks() error: %v", err)
	}
	// Map is returned but the key for this book should be absent.
	if authors, ok := result[book.ID]; ok && len(authors) > 0 {
		t.Errorf("expected no authors for book, got %d", len(authors))
	}
}
