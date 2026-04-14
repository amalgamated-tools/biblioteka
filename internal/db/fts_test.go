package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckFTSIntegrity_FreshDB(t *testing.T) {
	d := newTestDB(t)
	err := d.CheckFTSIntegrity(t.Context())
	require.NoError(t, err, "CheckFTSIntegrity() on fresh DB should succeed")
}

func TestCheckFTSIntegrity_MissingFTSTable(t *testing.T) {
	d := newTestDB(t)

	_, err := d.ExecContext(t.Context(), "DROP TABLE books_fts")
	require.NoError(t, err, "DROP TABLE books_fts")

	err = d.CheckFTSIntegrity(t.Context())
	require.Error(t, err, "CheckFTSIntegrity() should fail when books_fts is missing")
}
func TestRebuildFTS_FreshDB(t *testing.T) {
	d := newTestDB(t)
	err := d.RebuildFTS(t.Context())
	require.NoError(t, err, "RebuildFTS() on fresh DB should succeed")
}

func TestRebuildFTS_AfterBooksInserted(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook()")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err, "CreateBook()")

	err = d.RebuildFTS(t.Context())
	require.NoError(t, err, "RebuildFTS() after inserts should succeed")

	// Ensure search still works after rebuild.
	books, total, err := d.SearchBooks(t.Context(), "Foundation", 10, 0)
	require.NoError(t, err, "SearchBooks() after rebuild")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, "Foundation", books[0].Title)
}

func TestCheckFTSIntegrity_AfterRebuild(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Neuromancer"})
	require.NoError(t, err, "CreateBook()")

	err = d.RebuildFTS(t.Context())
	require.NoError(t, err, "RebuildFTS()")

	err = d.CheckFTSIntegrity(t.Context())
	require.NoError(t, err, "CheckFTSIntegrity() after rebuild should succeed")
}
