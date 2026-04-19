package calibre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// previewLimit is the maximum number of books returned in a Preview.
const previewLimit = 25

// Sentinel errors returned by WebImport for error classification.
var (
	// ErrLibraryNotFound is returned when the specified library_id does not exist.
	ErrLibraryNotFound = errors.New("library not found")

	// ErrLoadCalibreBooks is returned when the Calibre database cannot be read.
	ErrLoadCalibreBooks = errors.New("could not read Calibre books")
)

// PreviewSeriesEntry is one series membership included in a preview book.
type PreviewSeriesEntry struct {
	Name     string  `json:"name"`
	Position float64 `json:"position"`
}

// PreviewBook holds the metadata for a single Calibre book as shown in the
// web import preview.
type PreviewBook struct {
	CalibreID       int64                `json:"calibre_id"`
	Title           string               `json:"title"`
	Authors         []string             `json:"authors"`
	Series          []PreviewSeriesEntry `json:"series"`
	Publisher       string               `json:"publisher,omitempty"`
	PublicationDate string               `json:"publication_date,omitempty"`
	ISBN13          string               `json:"isbn13,omitempty"`
	ISBN10          string               `json:"isbn10,omitempty"`
	Formats         []string             `json:"formats"`
}

// Preview holds the result of parsing a Calibre metadata.db for display in
// the web import wizard.
type Preview struct {
	// Total is the total number of books found in the Calibre database.
	Total int `json:"total"`

	// Books contains up to previewLimit books for display. The caller should
	// use Total to inform the user about books beyond the preview window.
	Books []PreviewBook `json:"books"`
}

// LoadPreview reads all books from calibreDB and returns a preview summary.
// It does not write to the Biblioteka database.
func LoadPreview(ctx context.Context, calibreDB *DB) (*Preview, error) {
	books, err := calibreDB.LoadBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("load calibre books: %w", err)
	}

	limit := min(len(books), previewLimit)

	preview := &Preview{
		Total: len(books),
		Books: make([]PreviewBook, 0, limit),
	}

	for _, book := range books[:limit] {
		preview.Books = append(preview.Books, toPreviewBook(&book))
	}

	return preview, nil
}

// toPreviewBook converts a calibre.Book to a PreviewBook.
func toPreviewBook(book *Book) PreviewBook {
	authors := book.Authors
	if authors == nil {
		authors = []string{}
	}

	series := make([]PreviewSeriesEntry, 0, len(book.Series))
	for _, se := range book.Series {
		series = append(series, PreviewSeriesEntry(se))
	}

	formats := make([]string, 0, len(book.Formats))
	for _, f := range book.Formats {
		formats = append(formats, strings.ToLower(f.FormatCode))
	}

	pb := PreviewBook{
		CalibreID: book.CalibreID,
		Title:     book.Title,
		Authors:   authors,
		Series:    series,
		Publisher: book.Publisher,
		Formats:   formats,
	}

	if !book.Pubdate.IsZero() {
		pb.PublicationDate = book.Pubdate.Format("2006-01-02")
	}

	pb.ISBN13 = book.Identifiers["isbn13"]
	pb.ISBN10 = book.Identifiers["isbn10"]

	return pb
}

// WebImportOptions configures a web-UI-driven Calibre metadata import.
// Unlike ImportOptions, no library path is required: file records are not
// created — only book metadata (title, authors, series, identifiers) is
// imported.
type WebImportOptions struct {
	// LibraryID optionally associates every imported book with a Biblioteka
	// library. When empty, books are imported without a library association.
	LibraryID string
}

// WebImport reads Calibre books from calibreDB and imports their metadata into
// biblDB without creating any file records. Books are deduplicated by ISBN-13,
// ISBN-10, ASIN, or Goodreads ID when those fields are present. Books with no
// recognised external identifier are always imported.
func WebImport(ctx context.Context, biblDB *db.DB, calibreDB *DB, opts WebImportOptions) (*ImportResult, error) {
	if opts.LibraryID != "" {
		if _, err := biblDB.GetLibrary(ctx, opts.LibraryID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: %q", ErrLibraryNotFound, opts.LibraryID)
			}
			return nil, fmt.Errorf("validate library %q: %w", opts.LibraryID, err)
		}
	}

	books, err := calibreDB.LoadBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLoadCalibreBooks, err)
	}

	result := &ImportResult{Total: len(books)}
	slog.InfoContext(ctx, "calibre: starting web import",
		slog.Int(otelkeys.BookCount, len(books)),
	)

	for i := range books {
		book := &books[i]
		imported, importErr := webImportBook(ctx, biblDB, book, opts)
		if importErr != nil {
			slog.WarnContext(ctx, "calibre: failed to web-import book",
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

	slog.InfoContext(ctx, "calibre: web import complete",
		slog.Int(otelkeys.BookCount, result.Total),
		slog.Int(otelkeys.Imported, result.Imported),
		slog.Int(otelkeys.Skipped, result.Skipped),
		slog.Int(otelkeys.ErrorCount, result.Errors),
	)

	return result, nil
}

// webImportBook imports a single Calibre book as metadata only (no file
// records). It returns (true, nil) on a successful write, (false, nil) when
// the book is deduplicated by an existing external identifier, and
// (false, err) on a hard error.
func webImportBook(ctx context.Context, biblDB *db.DB, book *Book, opts WebImportOptions) (bool, error) {
	input := buildBookInput(book)

	// Deduplicate: if a book with the same ISBN-13, ISBN-10, ASIN or Goodreads
	// ID already exists, skip it to keep the import idempotent.
	existing, err := biblDB.FindBookByExternalID(ctx, input.ISBN13, input.ISBN10, input.ASIN, input.GoodreadsID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("check duplicate for %q: %w", book.Title, err)
	}
	if existing != nil {
		slog.DebugContext(ctx, "calibre: skipping already-indexed book",
			slog.Int64(otelkeys.CalibreID, book.CalibreID),
			slog.String(otelkeys.Title, book.Title),
		)
		return false, nil
	}

	biblBook, err := biblDB.CreateBook(ctx, input)
	if err != nil {
		return false, fmt.Errorf("create book %q: %w", book.Title, err)
	}

	slog.DebugContext(ctx, "calibre: created book (web import)",
		slog.String(otelkeys.BookID, biblBook.ID),
		slog.Int64(otelkeys.CalibreID, book.CalibreID),
		slog.String(otelkeys.Title, biblBook.Title),
	)

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
