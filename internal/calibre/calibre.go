// Package calibre provides a read-only client for Calibre metadata.db files
// and an importer that copies books, authors, series, and file records into a
// Biblioteka database.
package calibre

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // register the SQLite driver

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// DB is a read-only connection to a Calibre metadata.db SQLite database.
type DB struct {
	db *sql.DB
}

// Open opens the Calibre metadata.db at path in read-only mode. The caller
// must call Close when done.
func Open(path string) (*DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat calibre db %q: %w", path, err)
	}
	dsn := fmt.Sprintf("file:%s?mode=ro", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open calibre db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping calibre db: %w", err)
	}
	return &DB{db: sqlDB}, nil
}

// Close releases the database connection.
func (c *DB) Close() error {
	return c.db.Close()
}

// calibreSentinelYear is Calibre's magic year for "no publication date set"
// (the sentinel date is 0101-01-01, i.e. 1 January of year 101 AD).
const calibreSentinelYear = 101

var calibreDateLayouts = []string{
	"2006-01-02T15:04:05-07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02T15:04:05+00:00",
	"2006-01-02 15:04:05+00:00",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// parseCalibreDate parses a Calibre datetime string. Returns a zero Time on
// failure. Calibre uses the sentinel date "0101-01-01" for books with no
// publication date; callers should only use results whose year is greater
// than calibreSentinelYear.
func parseCalibreDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range calibreDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// Book holds all metadata for a single Calibre book, including its authors,
// series memberships, file formats, and external identifiers.
type Book struct {
	CalibreID   int64
	Title       string
	Path        string            // relative path within the library root
	SeriesIndex float64           // position within any linked series
	Pubdate     time.Time         // zero when no meaningful date is set
	Authors     []string          // author names in display order
	Series      []SeriesEntry     // series memberships
	Publisher   string            // empty when no publisher is set
	Description string            // empty when no comment is set
	Formats     []Format          // file formats available for this book
	Identifiers map[string]string // identifier type → value (e.g. "isbn" → "...")
	Language    string            // ISO 639 language code; empty when not set
}

// SeriesEntry represents one of a book's series memberships.
type SeriesEntry struct {
	Name     string
	Position float64
}

// Format represents a single file stored in the Calibre library.
type Format struct {
	FormatCode       string // uppercase extension, e.g. "EPUB"
	Name             string // filename without extension
	UncompressedSize int64
}

// FileName returns the filename including the lowercase extension.
func (f Format) FileName() string {
	return f.Name + "." + strings.ToLower(f.FormatCode)
}

// FilePath returns the absolute path for this format file.
func (f Format) FilePath(libraryRoot, bookPath string) string {
	return filepath.Join(libraryRoot, bookPath, f.FileName())
}

// rawBook holds the scalar columns read from Calibre's books table.
type rawBook struct {
	id          int64
	title       string
	path        string
	seriesIndex float64
	pubdate     time.Time
}

// LoadBooks reads all books and their associated metadata from the Calibre
// database in a single pass.
func (c *DB) LoadBooks(ctx context.Context) ([]Book, error) {
	slog.DebugContext(ctx, "calibre: loading books")

	raws, err := c.loadRawBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("load raw books: %w", err)
	}
	if len(raws) == 0 {
		return nil, nil
	}

	authors, err := c.loadBookAuthors(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book authors: %w", err)
	}

	seriesEntries, err := c.loadBookSeriesEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book series: %w", err)
	}

	publishers, err := c.loadBookPublishers(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book publishers: %w", err)
	}

	comments, err := c.loadBookComments(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book comments: %w", err)
	}

	formats, err := c.loadBookFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book formats: %w", err)
	}

	identifiers, err := c.loadBookIdentifiers(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book identifiers: %w", err)
	}

	languages, err := c.loadBookLanguages(ctx)
	if err != nil {
		return nil, fmt.Errorf("load book languages: %w", err)
	}

	books := make([]Book, 0, len(raws))
	for _, rb := range raws {
		book := Book{
			CalibreID:   rb.id,
			Title:       rb.title,
			Path:        rb.path,
			SeriesIndex: rb.seriesIndex,
			Authors:     authors[rb.id],
			Series:      seriesEntries[rb.id],
			Publisher:   publishers[rb.id],
			Description: comments[rb.id],
			Formats:     formats[rb.id],
			Identifiers: identifiers[rb.id],
			Language:    languages[rb.id],
		}
		// Year ≤ calibreSentinelYear is Calibre's sentinel for "no date set" (0101-01-01).
		if !rb.pubdate.IsZero() && rb.pubdate.Year() > calibreSentinelYear {
			book.Pubdate = rb.pubdate
		}
		books = append(books, book)
	}

	slog.DebugContext(ctx, "calibre: books loaded", slog.Int(otelkeys.BookCount, len(books)))
	return books, nil
}

func (c *DB) loadRawBooks(ctx context.Context) ([]rawBook, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT id, title, path, series_index, pubdate FROM books ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []rawBook
	for rows.Next() {
		var rb rawBook
		var pubdateStr string
		if err := rows.Scan(&rb.id, &rb.title, &rb.path, &rb.seriesIndex, &pubdateStr); err != nil {
			return nil, err
		}
		rb.pubdate = parseCalibreDate(pubdateStr)
		books = append(books, rb)
	}
	return books, rows.Err()
}

type bookIdentifier struct {
	typ string
	val string
}

func collectBookMap[V any, M any](
	ctx context.Context,
	db *sql.DB,
	query string,
	scan func(*sql.Rows) (int64, V, error),
	collect func(map[int64]M, int64, V),
) (map[int64]M, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]M)
	for rows.Next() {
		bookID, value, err := scan(rows)
		if err != nil {
			return nil, err
		}
		collect(result, bookID, value)
	}

	return result, rows.Err()
}

func (c *DB) loadBookAuthors(ctx context.Context) (map[int64][]string, error) {
	return collectBookMap(ctx, c.db,
		`SELECT bal.book, a.name
		 FROM authors a
		 INNER JOIN books_authors_link bal ON bal.author = a.id
		 ORDER BY bal.book, bal.id`,
		func(rows *sql.Rows) (int64, string, error) {
			var bookID int64
			var name string
			if err := rows.Scan(&bookID, &name); err != nil {
				return 0, "", err
			}
			return bookID, name, nil
		},
		func(result map[int64][]string, bookID int64, name string) {
			result[bookID] = append(result[bookID], name)
		},
	)
}

func (c *DB) loadBookSeriesEntries(ctx context.Context) (map[int64][]SeriesEntry, error) {
	return collectBookMap(ctx, c.db,
		`SELECT bsl.book, s.name, b.series_index
		 FROM series s
		 INNER JOIN books_series_link bsl ON bsl.series = s.id
		 INNER JOIN books b ON b.id = bsl.book
		 ORDER BY bsl.book, s.name`,
		func(rows *sql.Rows) (int64, SeriesEntry, error) {
			var bookID int64
			var entry SeriesEntry
			if err := rows.Scan(&bookID, &entry.Name, &entry.Position); err != nil {
				return 0, SeriesEntry{}, err
			}
			return bookID, entry, nil
		},
		func(result map[int64][]SeriesEntry, bookID int64, entry SeriesEntry) {
			result[bookID] = append(result[bookID], entry)
		},
	)
}

func (c *DB) loadBookPublishers(ctx context.Context) (map[int64]string, error) {
	return collectBookMap(ctx, c.db,
		`SELECT bpl.book, p.name
		 FROM publishers p
		 INNER JOIN books_publishers_link bpl ON bpl.publisher = p.id
		 ORDER BY bpl.book, bpl.id`,
		func(rows *sql.Rows) (int64, string, error) {
			var bookID int64
			var name string
			if err := rows.Scan(&bookID, &name); err != nil {
				return 0, "", err
			}
			return bookID, name, nil
		},
		func(result map[int64]string, bookID int64, name string) {
			// Keep the first publisher when a book has multiple (unusual in practice).
			if _, exists := result[bookID]; !exists {
				result[bookID] = name
			}
		},
	)
}

func (c *DB) loadBookComments(ctx context.Context) (map[int64]string, error) {
	return collectBookMap(ctx, c.db,
		`SELECT book, text FROM comments`,
		func(rows *sql.Rows) (int64, string, error) {
			var bookID int64
			var text string
			if err := rows.Scan(&bookID, &text); err != nil {
				return 0, "", err
			}
			return bookID, text, nil
		},
		func(result map[int64]string, bookID int64, text string) {
			result[bookID] = text
		},
	)
}

func (c *DB) loadBookFormats(ctx context.Context) (map[int64][]Format, error) {
	return collectBookMap(ctx, c.db,
		`SELECT book, format, name, uncompressed_size FROM data ORDER BY book, format`,
		func(rows *sql.Rows) (int64, Format, error) {
			var bookID int64
			var format Format
			if err := rows.Scan(&bookID, &format.FormatCode, &format.Name, &format.UncompressedSize); err != nil {
				return 0, Format{}, err
			}
			return bookID, format, nil
		},
		func(result map[int64][]Format, bookID int64, format Format) {
			result[bookID] = append(result[bookID], format)
		},
	)
}

func (c *DB) loadBookIdentifiers(ctx context.Context) (map[int64]map[string]string, error) {
	return collectBookMap(ctx, c.db,
		`SELECT book, type, val FROM identifiers ORDER BY book`,
		func(rows *sql.Rows) (int64, bookIdentifier, error) {
			var bookID int64
			var identifier bookIdentifier
			if err := rows.Scan(&bookID, &identifier.typ, &identifier.val); err != nil {
				return 0, bookIdentifier{}, err
			}
			return bookID, identifier, nil
		},
		func(result map[int64]map[string]string, bookID int64, identifier bookIdentifier) {
			if result[bookID] == nil {
				result[bookID] = make(map[string]string)
			}
			result[bookID][identifier.typ] = identifier.val
		},
	)
}

func (c *DB) loadBookLanguages(ctx context.Context) (map[int64]string, error) {
	result, err := collectBookMap(ctx, c.db,
		`SELECT bll.book, l.lang_code
		 FROM languages l
		 INNER JOIN books_languages_link bll ON bll.lang_code = l.id
		 ORDER BY bll.book, bll.item_order`,
		func(rows *sql.Rows) (int64, string, error) {
			var bookID int64
			var langCode string
			if err := rows.Scan(&bookID, &langCode); err != nil {
				return 0, "", err
			}
			return bookID, langCode, nil
		},
		func(result map[int64]string, bookID int64, langCode string) {
			// Retain only the primary language (lowest item_order) per book.
			if _, exists := result[bookID]; !exists {
				result[bookID] = langCode
			}
		},
	)
	if err != nil {
		// Older Calibre databases may not have the languages tables. The SQLite
		// driver does not expose a typed error for "no such table", so we fall
		// back to string matching as a last resort.
		if strings.Contains(err.Error(), "no such table") {
			return make(map[int64]string), nil
		}
		return nil, err
	}
	return result, nil
}
