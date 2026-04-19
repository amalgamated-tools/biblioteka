package db

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookRelations holds the related entities for a single book: its authors,
// files, series entries, and tags. Use [DB.LoadBookRelations] to populate it.
type BookRelations struct {
	Authors []Author
	Files   []BookFile
	Series  []BookSeriesEntry
	Tags    []Tag
}

// LoadBookRelations fetches the authors, files, series entries, and tags for a
// single book by delegating to the batch APIs ([DB.GetAuthorsForBooks],
// [DB.GetFilesForBooks], [DB.GetSeriesForBooks], [DB.GetTagsForBooks]). This
// keeps the query pattern consistent with the Kobo sync endpoint while
// providing a convenient single-book wrapper for use in metadata and detail
// endpoints.
//
// The four queries run concurrently to reduce latency on the book detail
// endpoint; SQLite WAL mode allows concurrent readers on separate connections.
func (d *DB) LoadBookRelations(ctx context.Context, bookID string) (*BookRelations, error) {
	slog.DebugContext(ctx, "db: loading book relations", slog.String(otelkeys.BookID, bookID))

	ids := []string{bookID}

	var (
		authorsByBook map[string][]Author
		filesByBook   map[string][]BookFile
		seriesByBook  map[string][]BookSeriesEntry
		tagsByBook    map[string][]Tag
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		authorsByBook, err = d.GetAuthorsForBooks(gctx, ids)
		if err != nil {
			return fmt.Errorf("load book authors: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		filesByBook, err = d.GetFilesForBooks(gctx, ids)
		if err != nil {
			return fmt.Errorf("load book files: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		seriesByBook, err = d.GetSeriesForBooks(gctx, ids)
		if err != nil {
			return fmt.Errorf("load book series: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		tagsByBook, err = d.GetTagsForBooks(gctx, ids)
		if err != nil {
			return fmt.Errorf("load book tags: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &BookRelations{
		Authors: authorsByBook[bookID],
		Files:   filesByBook[bookID],
		Series:  seriesByBook[bookID],
		Tags:    tagsByBook[bookID],
	}, nil
}
