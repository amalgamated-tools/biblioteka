package calibre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/ptrutil"
)

// ImportOptions configures a Calibre library import.
type ImportOptions struct {
	// LibraryPath is the absolute path to the root of the Calibre library.
	// It must contain a metadata.db file.
	LibraryPath string

	// LibraryID optionally associates every imported book with an existing
	// Biblioteka library. When empty, books are imported without a library
	// association.
	LibraryID string
}

// ImportResult summarises the outcome of an import run.
type ImportResult struct {
	// Total is the total number of Calibre books examined.
	Total int

	// Imported is the number of books successfully written to the database.
	Imported int

	// Skipped is the number of books not written because they were already
	// present (detected by file path).
	Skipped int

	// Errors is the number of books that could not be imported due to errors.
	Errors int
}

// Import reads the Calibre metadata.db at opts.LibraryPath/metadata.db and
// copies books, authors, series, and file records into biblDB. Each book is
// deduplicated by file path: if any of its format file paths already exist in
// book_files, the book is skipped. Per-book errors are logged and counted
// without aborting the import of remaining books.
func Import(ctx context.Context, biblDB *db.DB, opts ImportOptions) (*ImportResult, error) {
	metaPath := filepath.Join(opts.LibraryPath, "metadata.db")
	slog.InfoContext(ctx, "calibre: starting import",
		slog.String(otelkeys.Path, metaPath),
		slog.String(otelkeys.LibraryID, opts.LibraryID),
	)

	calibreDB, err := Open(metaPath)
	if err != nil {
		return nil, fmt.Errorf("open calibre library at %q: %w", metaPath, err)
	}
	defer func() {
		if cerr := calibreDB.Close(); cerr != nil {
			slog.WarnContext(ctx, "calibre: failed to close db", slog.Any(otelkeys.Error, cerr))
		}
	}()

	return runImport(ctx, biblDB, calibreDB, opts)
}

// runImport is the internal implementation of Import, split out so tests can
// inject a pre-populated calibre.DB directly.
func runImport(ctx context.Context, biblDB *db.DB, calibreDB *DB, opts ImportOptions) (*ImportResult, error) {
	books, err := calibreDB.LoadBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("load calibre books: %w", err)
	}

	result := &ImportResult{Total: len(books)}
	slog.InfoContext(ctx, "calibre: loaded books",
		slog.Int(otelkeys.BookCount, len(books)),
	)

	for i := range books {
		book := &books[i]
		imported, importErr := importBook(ctx, biblDB, book, opts)
		if importErr != nil {
			slog.WarnContext(ctx, "calibre: failed to import book",
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

	slog.InfoContext(ctx, "calibre: import complete",
		slog.Int(otelkeys.BookCount, result.Total),
		slog.Int(otelkeys.Imported, result.Imported),
		slog.Int(otelkeys.Skipped, result.Skipped),
		slog.Int(otelkeys.ErrorCount, result.Errors),
	)
	return result, nil
}

// importBook imports a single Calibre book. It returns (true, nil) when the
// book is successfully written, (false, nil) when it is skipped as a
// duplicate, and (false, err) on a hard error.
func importBook(ctx context.Context, biblDB *db.DB, book *Book, opts ImportOptions) (bool, error) {
	// Books with no file formats cannot be deduplicated by file path, so they
	// would be re-imported on every run. Skip them to keep the import
	// idempotent; users can add metadata-only books manually if needed.
	if len(book.Formats) == 0 {
		slog.DebugContext(ctx, "calibre: skipping format-less book",
			slog.Int64(otelkeys.CalibreID, book.CalibreID),
			slog.String(otelkeys.Title, book.Title),
		)
		return false, nil
	}

	// Deduplicate by file path: skip the entire book if any format is already indexed.
	for _, f := range book.Formats {
		path := f.FilePath(opts.LibraryPath, book.Path)
		_, err := biblDB.GetBookFileByPath(ctx, path)
		if err == nil {
			slog.DebugContext(ctx, "calibre: skipping already-indexed book",
				slog.Int64(otelkeys.CalibreID, book.CalibreID),
				slog.String(otelkeys.Title, book.Title),
				slog.String(otelkeys.FilePath, path),
			)
			return false, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("check duplicate path %q: %w", path, err)
		}
	}

	input := buildBookInput(book)
	biblBook, err := biblDB.CreateBook(ctx, input)
	if err != nil {
		return false, fmt.Errorf("create book %q: %w", book.Title, err)
	}

	slog.DebugContext(ctx, "calibre: created book",
		slog.String(otelkeys.BookID, biblBook.ID),
		slog.Int64(otelkeys.CalibreID, book.CalibreID),
		slog.String(otelkeys.Title, biblBook.Title),
	)

	// Register each file format.
	for _, f := range book.Formats {
		path := f.FilePath(opts.LibraryPath, book.Path)
		fileType := strings.ToLower(f.FormatCode)
		if _, fileErr := biblDB.CreateBookFile(
			ctx, biblBook.ID, fileType, f.FileName(), f.UncompressedSize, nil, path,
		); fileErr != nil {
			slog.WarnContext(ctx, "calibre: failed to create book file",
				slog.String(otelkeys.BookID, biblBook.ID),
				slog.String(otelkeys.FileName, f.FileName()),
				slog.Any(otelkeys.Error, fileErr),
			)
		}
	}

	// Link authors — best-effort.
	if len(book.Authors) > 0 {
		authorIDs := make([]string, 0, len(book.Authors))
		for _, name := range book.Authors {
			author, authorErr := biblDB.FindOrCreateAuthor(ctx, name)
			if authorErr != nil {
				slog.WarnContext(ctx, "calibre: failed to find or create author",
					slog.String(otelkeys.BookID, biblBook.ID),
					slog.String(otelkeys.Author, name),
					slog.Any(otelkeys.Error, authorErr),
				)
				continue
			}
			authorIDs = append(authorIDs, author.ID)
		}
		if len(authorIDs) > 0 {
			if err := biblDB.SetBookAuthors(ctx, biblBook.ID, authorIDs); err != nil {
				slog.WarnContext(ctx, "calibre: failed to set book authors",
					slog.String(otelkeys.BookID, biblBook.ID),
					slog.Any(otelkeys.Error, err),
				)
			}
		}
	}

	// Link series — best-effort.
	if len(book.Series) > 0 {
		seriesInputs := make([]db.BookSeriesInput, 0, len(book.Series))
		for _, se := range book.Series {
			s, seriesErr := biblDB.FindOrCreateSeries(ctx, se.Name)
			if seriesErr != nil {
				slog.WarnContext(ctx, "calibre: failed to find or create series",
					slog.String(otelkeys.BookID, biblBook.ID),
					slog.String(otelkeys.Name, se.Name),
					slog.Any(otelkeys.Error, seriesErr),
				)
				continue
			}
			pos := se.Position
			seriesInputs = append(seriesInputs, db.BookSeriesInput{
				SeriesID: s.ID,
				Position: &pos,
			})
		}
		if len(seriesInputs) > 0 {
			if err := biblDB.SetBookSeries(ctx, biblBook.ID, seriesInputs); err != nil {
				slog.WarnContext(ctx, "calibre: failed to set book series",
					slog.String(otelkeys.BookID, biblBook.ID),
					slog.Any(otelkeys.Error, err),
				)
			}
		}
	}

	// Associate with a Biblioteka library — best-effort.
	if opts.LibraryID != "" {
		if err := biblDB.AddBookToLibrary(ctx, opts.LibraryID, biblBook.ID); err != nil {
			slog.WarnContext(ctx, "calibre: failed to add book to library",
				slog.String(otelkeys.BookID, biblBook.ID),
				slog.String(otelkeys.LibraryID, opts.LibraryID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	return true, nil
}

// buildBookInput maps a Calibre book to a Biblioteka BookInput.
func buildBookInput(book *Book) db.BookInput {
	input := db.BookInput{
		Title:     book.Title,
		Publisher: ptrutil.NilIfZero(book.Publisher),
		Language:  ptrutil.NilIfZero(book.Language),
	}

	if book.Description != "" {
		desc := book.Description
		input.Description = &desc
	}

	if !book.Pubdate.IsZero() {
		date := book.Pubdate.Format("2006-01-02")
		input.PublicationDate = &date
	}

	// Map Calibre identifier types to Biblioteka BookInput fields.
	// ISBN types are applied in priority order (isbn13 > isbn10 > isbn)
	// rather than map iteration order to ensure deterministic results when
	// multiple ISBN identifiers are present.
	for _, typ := range []string{"isbn13", "isbn10", "isbn"} {
		val, ok := book.Identifiers[typ]
		if !ok || val == "" {
			continue
		}
		normalized := exif.NormalizeISBN(val)
		switch len(normalized) {
		case 10:
			input.ISBN10 = ptrutil.NilIfZero(normalized)
		case 13:
			input.ISBN13 = ptrutil.NilIfZero(normalized)
		}
	}
	for typ, val := range book.Identifiers {
		if val == "" {
			continue
		}
		switch strings.ToLower(typ) {
		case "asin", "mobi-asin":
			input.ASIN = ptrutil.NilIfZero(strings.TrimSpace(val))
		case "goodreads", "goodreads-id":
			input.GoodreadsID = ptrutil.NilIfZero(strings.TrimSpace(val))
		case "google", "google-id", "googlebooks":
			input.GoogleBooksID = ptrutil.NilIfZero(strings.TrimSpace(val))
		case "hardcover", "hardcover-id":
			input.HardcoverID = ptrutil.NilIfZero(strings.TrimSpace(val))
		}
	}

	return input
}
