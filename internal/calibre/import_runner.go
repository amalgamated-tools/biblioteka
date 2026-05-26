package calibre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

type runBookImportOptions struct {
	libraryID string

	loadedBooksMsg string
	completeMsg    string
	importErrMsg   string

	libraryNotFoundErr func(libraryID string) error
	loadBooksErr       func(err error) error
	afterLoad          func(ctx context.Context, count int)
	importOne          func(ctx context.Context, book *Book) (bool, error)
}

func runBookImport(
	ctx context.Context,
	biblDB *db.DB,
	calibreDB *DB,
	opts runBookImportOptions,
) (*ImportResult, error) {
	if opts.importOne == nil {
		panic("runBookImportOptions.importOne must not be nil")
	}

	if opts.libraryID != "" {
		if _, err := biblDB.GetLibrary(ctx, opts.libraryID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if opts.libraryNotFoundErr != nil {
					return nil, opts.libraryNotFoundErr(opts.libraryID)
				}

				return nil, fmt.Errorf("library %q not found", opts.libraryID)
			}

			return nil, fmt.Errorf("validate library %q: %w", opts.libraryID, err)
		}
	}

	books, err := calibreDB.LoadBooks(ctx)
	if err != nil {
		if opts.loadBooksErr != nil {
			return nil, opts.loadBooksErr(err)
		}

		return nil, fmt.Errorf("load calibre books: %w", err)
	}

	result := &ImportResult{Total: len(books)}
	slog.InfoContext(ctx, opts.loadedBooksMsg, slog.Int(otelkeys.BookCount, len(books)))

	if opts.afterLoad != nil {
		opts.afterLoad(ctx, len(books))
	}

	for i := range books {
		book := &books[i]
		imported, importErr := opts.importOne(ctx, book)
		if importErr != nil {
			slog.WarnContext(ctx, opts.importErrMsg,
				slog.Int64(otelkeys.CalibreID, book.CalibreID),
				slog.String(otelkeys.Title, book.Title),
				slog.Any(otelkeys.Error, importErr),
			)
			result.Errors++

			continue
		}

		if imported {
			result.Imported++
		} else {
			result.Skipped++
		}
	}

	slog.InfoContext(ctx, opts.completeMsg,
		slog.Int(otelkeys.BookCount, result.Total),
		slog.Int(otelkeys.Imported, result.Imported),
		slog.Int(otelkeys.Skipped, result.Skipped),
		slog.Int(otelkeys.ErrorCount, result.Errors),
	)

	return result, nil
}
