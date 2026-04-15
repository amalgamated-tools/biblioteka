package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookRelations holds the related entities for a single book: its authors,
// files, and series entries. Use [DB.LoadBookRelations] to populate it.
type BookRelations struct {
	Authors []Author
	Files   []BookFile
	Series  []BookSeriesEntry
}

// LoadBookRelations fetches the authors, files, and series entries for a single
// book by delegating to the batch APIs ([DB.GetAuthorsForBooks],
// [DB.GetFilesForBooks], [DB.GetSeriesForBooks]). This keeps the query pattern
// consistent with the Kobo sync endpoint while providing a convenient
// single-book wrapper for use in metadata and detail endpoints.
func (d *DB) LoadBookRelations(ctx context.Context, bookID string) (*BookRelations, error) {
	slog.DebugContext(ctx, "loading book relations", slog.String(otelkeys.BookID, bookID))

	ids := []string{bookID}

	authorsByBook, err := d.GetAuthorsForBooks(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load book authors: %w", err)
	}

	filesByBook, err := d.GetFilesForBooks(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load book files: %w", err)
	}

	seriesByBook, err := d.GetSeriesForBooks(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load book series: %w", err)
	}

	return &BookRelations{
		Authors: authorsByBook[bookID],
		Files:   filesByBook[bookID],
		Series:  seriesByBook[bookID],
	}, nil
}
