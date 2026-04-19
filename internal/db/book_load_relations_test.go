package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadBookRelations_Empty(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Lonely Book"})
	require.NoError(t, err)

	rels, err := d.LoadBookRelations(t.Context(), book.ID)
	require.NoError(t, err)
	require.NotNil(t, rels)
	require.Empty(t, rels.Authors)
	require.Empty(t, rels.Files)
	require.Empty(t, rels.Series)
	require.Empty(t, rels.Tags)
}

func TestLoadBookRelations_WithRelations(t *testing.T) {
	d := newTestDB(t)

	// Create a book.
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Loaded Book"})
	require.NoError(t, err)

	// Add an author and link it.
	author, err := d.CreateAuthor(t.Context(), "Test Author", nil, nil, nil, nil)
	require.NoError(t, err)
	err = d.SetBookAuthors(t.Context(), book.ID, []string{author.ID})
	require.NoError(t, err)

	// Add a series and link it.
	series, err := d.CreateSeries(t.Context(), "Test Series", nil, nil, nil)
	require.NoError(t, err)
	pos := 2.0
	err = d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{
		{SeriesID: series.ID, Position: &pos},
	})
	require.NoError(t, err)

	// Add a file.
	_, err = d.CreateBookFile(t.Context(), book.ID, "epub", "book.epub", 1024, nil, filepath.Join(t.TempDir(), "book.epub"))
	require.NoError(t, err)

	// Add a tag and link it.
	tag, err := d.CreateTag(t.Context(), "fiction")
	require.NoError(t, err)
	err = d.SetBookTags(t.Context(), book.ID, []string{tag.ID})
	require.NoError(t, err)

	// Load relations.
	rels, err := d.LoadBookRelations(t.Context(), book.ID)
	require.NoError(t, err)
	require.NotNil(t, rels)

	require.Len(t, rels.Authors, 1)
	require.Equal(t, "Test Author", rels.Authors[0].Name)

	require.Len(t, rels.Series, 1)
	require.Equal(t, "Test Series", rels.Series[0].Series.Name)
	require.NotNil(t, rels.Series[0].Position)
	require.InDelta(t, 2.0, *rels.Series[0].Position, 0.001)

	require.Len(t, rels.Files, 1)
	require.Equal(t, "book.epub", rels.Files[0].FileName)

	require.Len(t, rels.Tags, 1)
	require.Equal(t, "fiction", rels.Tags[0].Name)
}
