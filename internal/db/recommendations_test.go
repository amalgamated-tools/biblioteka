package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- helpers ----

func createBookWithPublisher(t *testing.T, d *DB, title, publisher string) *Book {
	t.Helper()
	pub := publisher
	b, err := d.CreateBook(t.Context(), BookInput{Title: title, Publisher: &pub})
	require.NoError(t, err, "CreateBook(%q)", title)
	return b
}

// ---- GetRecommendations ----

func TestGetRecommendations_EmptyLibrary(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err, "GetRecommendations() error")
	require.Empty(t, books, "expected no books in empty library")
}

func TestGetRecommendations_NoHistory_ReturnsAllBooks(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	b1, err := d.CreateBook(t.Context(), BookInput{Title: "Book Alpha"})
	require.NoError(t, err)
	b2, err := d.CreateBook(t.Context(), BookInput{Title: "Book Beta"})
	require.NoError(t, err)

	// No reading history — all books score 0, both should appear as candidates.
	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, books, 2)
	ids := map[string]bool{books[0].ID: true, books[1].ID: true}
	require.True(t, ids[b1.ID], "b1 should be in results")
	require.True(t, ids[b2.ID], "b2 should be in results")
}

func TestGetRecommendations_ExcludesReadBooks(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	read, err := d.CreateBook(t.Context(), BookInput{Title: "Already Read"})
	require.NoError(t, err)
	unread, err := d.CreateBook(t.Context(), BookInput{Title: "Unread Book"})
	require.NoError(t, err)

	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, read.ID, StatusFinished, nil, nil, nil, nil)
	require.NoError(t, err)

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	ids := make([]string, len(books))
	for i, b := range books {
		ids[i] = b.ID
	}
	require.Contains(t, ids, unread.ID, "unread book should be in recommendations")
	require.NotContains(t, ids, read.ID, "already-read book must be excluded")
}

func TestGetRecommendations_AuthorOverlapScoresHigher(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	author, err := d.CreateAuthor(t.Context(), "Ursula K. Le Guin", nil, nil, nil, nil)
	require.NoError(t, err)

	// Book the user has already read, by that author.
	readBook, err := d.CreateBook(t.Context(), BookInput{Title: "The Left Hand of Darkness"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookAuthors(t.Context(), readBook.ID, []string{author.ID}))
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, readBook.ID, StatusFinished, nil, nil, nil, nil)
	require.NoError(t, err)

	// Same-author book — should score higher.
	sameAuthor, err := d.CreateBook(t.Context(), BookInput{Title: "The Dispossessed"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookAuthors(t.Context(), sameAuthor.ID, []string{author.ID}))

	// Unrelated book — no overlap.
	unrelated, err := d.CreateBook(t.Context(), BookInput{Title: "Moby Dick"})
	require.NoError(t, err)

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, sameAuthor.ID, books[0].ID, "same-author book should rank first")
	require.Equal(t, unrelated.ID, books[1].ID)
}

func TestGetRecommendations_SeriesContinuationScoresHighest(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	series, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err)

	pos1 := 1.0
	pos2 := 2.0

	// Book 1 in series: user has already read this.
	b1, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos1}}))
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, StatusFinished, nil, nil, nil, nil)
	require.NoError(t, err)

	// Book 2 in series: should score highest as series continuation.
	b2, err := d.CreateBook(t.Context(), BookInput{Title: "The Drawing of the Three"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos2}}))

	// Unrelated book.
	unrelated, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, b2.ID, books[0].ID, "series continuation should rank first")
	require.Equal(t, unrelated.ID, books[1].ID)
}

func TestGetRecommendations_SeriesContinuationOnlyNextBook(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	series, err := d.CreateSeries(t.Context(), "Wheel of Time", nil, nil, nil)
	require.NoError(t, err)

	pos1 := 1.0
	pos2 := 2.0
	pos3 := 3.0
	pos5 := 5.0

	// Book 1: user has read.
	b1, err := d.CreateBook(t.Context(), BookInput{Title: "Eye of the World"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos1}}))
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b1.ID, StatusFinished, nil, nil, nil, nil)
	require.NoError(t, err)

	// Book 2: immediate next — should get series bonus.
	b2, err := d.CreateBook(t.Context(), BookInput{Title: "The Great Hunt"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos2}}))

	// Book 3: NOT the immediate next — should NOT get series bonus.
	b3, err := d.CreateBook(t.Context(), BookInput{Title: "The Dragon Reborn"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookSeries(t.Context(), b3.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos3}}))

	// Book 5: much later — should NOT get series bonus.
	b5, err := d.CreateBook(t.Context(), BookInput{Title: "Fires of Heaven"})
	require.NoError(t, err)
	require.NoError(t, d.SetBookSeries(t.Context(), b5.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos5}}))

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, books, 3)
	// Book 2 should rank first due to series continuation bonus.
	require.Equal(t, b2.ID, books[0].ID, "only the immediate next book should get series bonus")
	// Books 3 and 5 should both be returned but without the bonus (same score, ordered by created_at DESC).
	remainingIDs := []string{books[1].ID, books[2].ID}
	require.Contains(t, remainingIDs, b3.ID)
	require.Contains(t, remainingIDs, b5.ID)
}

func TestGetRecommendations_PublisherOverlapAddsScore(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	// Book the user has read from "Ace Books".
	readBook := createBookWithPublisher(t, d, "Foundation", "Ace Books")
	_, err := d.UpsertKoboReadingState(t.Context(), user.ID, readBook.ID, "Finished", nil, nil, nil, nil)
	require.NoError(t, err)

	// Same publisher candidate — should score +1 above unrelated.
	samePublisher := createBookWithPublisher(t, d, "Foundation and Empire", "Ace Books")
	// Different publisher candidate.
	otherPublisher := createBookWithPublisher(t, d, "Neuromancer", "Ace Books 2")

	// Make otherPublisher created later so it normally sorts first with no score.
	_ = otherPublisher

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, samePublisher.ID, books[0].ID, "same-publisher book should score higher")
}

func TestGetRecommendations_LimitRespected(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	for i := range 5 {
		_, err := d.CreateBook(t.Context(), BookInput{Title: "Book " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	books, err := d.GetRecommendations(t.Context(), user.ID, 3, 0)
	require.NoError(t, err)
	require.Len(t, books, 3, "limit should be respected")
}

func TestGetRecommendations_ReadyToReadIncluded(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	// A book the user has in "ReadyToRead" status — should NOT be treated as
	// read/reading, so it remains a candidate.
	b, err := d.CreateBook(t.Context(), BookInput{Title: "Want To Read"})
	require.NoError(t, err)
	_, err = d.UpsertKoboReadingState(t.Context(), user.ID, b.ID, StatusReadyToRead, nil, nil, nil, nil)
	require.NoError(t, err)

	books, err := d.GetRecommendations(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	ids := make([]string, len(books))
	for i, bk := range books {
		ids[i] = bk.ID
	}
	require.Contains(t, ids, b.ID, "ReadyToRead books should appear as candidates")
}

func TestGetRecommendations_OffsetRespected(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	for i := range 5 {
		_, err := d.CreateBook(t.Context(), BookInput{Title: "Book " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	all, err := d.GetRecommendations(t.Context(), user.ID, 5, 0)
	require.NoError(t, err)
	require.Len(t, all, 5)

	// offset=2 should return the last 3 books (indices 2–4 of the full set).
	paged, err := d.GetRecommendations(t.Context(), user.ID, 5, 2)
	require.NoError(t, err)
	require.Len(t, paged, 3, "offset=2 should skip the first 2 results")
	require.Equal(t, all[2].ID, paged[0].ID, "first paged result should match third overall result")

	// offset beyond total should return empty.
	beyond, err := d.GetRecommendations(t.Context(), user.ID, 5, 10)
	require.NoError(t, err)
	require.Empty(t, beyond, "offset beyond total should return no results")
}
