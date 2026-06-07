package db

import (
	"context"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookFile represents a row in the book_files table.
type BookFile struct {
	ID            string    `json:"id"`
	BookID        string    `json:"book_id"`
	FileType      string    `json:"file_type"`
	FileName      string    `json:"file_name"`
	FileSize      int64     `json:"file_size"`
	FileHash      *string   `json:"file_hash"`
	FilePath      string    `json:"file_path"`
	DownloadCount int64     `json:"download_count"`
	CreatedAt     Timestamp `json:"created_at"`
	UpdatedAt     Timestamp `json:"updated_at"`
}

const bookFileColumns = `id, book_id, file_type, file_name, file_size, file_hash, file_path, download_count, created_at, updated_at`

// scanBookFile scans a book file row into a BookFile struct.
func scanBookFile(row interface{ Scan(...any) error }) (*BookFile, error) {
	return scanRow(row, func(bf *BookFile) []any {
		return []any{&bf.ID, &bf.BookID, &bf.FileType, &bf.FileName, &bf.FileSize, &bf.FileHash, &bf.FilePath, &bf.DownloadCount, &bf.CreatedAt, &bf.UpdatedAt}
	})
}

// bookFileColumnsWithPrefix returns book_files columns with a table alias prefix.
func bookFileColumnsWithPrefix(prefix string) string {
	cols := strings.Split(bookFileColumns, ",")
	for i, c := range cols {
		cols[i] = prefix + strings.TrimSpace(c)
	}
	return strings.Join(cols, ", ")
}

// CreateBookFile inserts a new book file record and returns it.
func (d *DB) CreateBookFile(ctx context.Context, bookID, fileType, fileName string, fileSize int64, fileHash *string, filePath string) (*BookFile, error) {
	slog.DebugContext(ctx, "db: creating book file",
		slog.String(otelkeys.BookID, bookID),
		slog.String(otelkeys.FileName, fileName),
	)
	bf, err := scanBookFile(d.QueryRowContext(ctx,
		`INSERT INTO book_files (book_id, file_type, file_name, file_size, file_hash, file_path) VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+bookFileColumns,
		bookID, fileType, fileName, fileSize, fileHash, filePath,
	))
	if err != nil {
		return nil, err
	}
	return bf, nil
}

// GetBookFile returns a book file by ID, or sql.ErrNoRows if not found.
func (d *DB) GetBookFile(ctx context.Context, id string) (*BookFile, error) {
	slog.DebugContext(ctx, "db: fetching book file", slog.String(otelkeys.BookFileID, id))
	return scanBookFile(d.QueryRowContext(ctx,
		`SELECT `+bookFileColumns+` FROM book_files WHERE id = $1`,
		id,
	))
}

// ListBookFiles returns all files for a given book.
func (d *DB) ListBookFiles(ctx context.Context, bookID string) ([]BookFile, error) {
	slog.DebugContext(ctx, "db: listing book files", slog.String(otelkeys.BookID, bookID))
	orderBy := d.dialectOrderBy("file_name", "ASC")
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookFileColumns+` FROM book_files WHERE book_id = $1 `+orderBy,
		bookID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBookFile)
}

// GetBookFileByPath returns a book file by its file path, or sql.ErrNoRows if not found.
func (d *DB) GetBookFileByPath(ctx context.Context, filePath string) (*BookFile, error) {
	slog.DebugContext(ctx, "db: fetching book file by path", slog.String(otelkeys.Path, filePath))
	return scanBookFile(d.QueryRowContext(ctx,
		`SELECT `+bookFileColumns+` FROM book_files WHERE file_path = $1`,
		filePath,
	))
}

// DeleteBookFile removes a book file by ID.
func (d *DB) DeleteBookFile(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting book file", slog.String(otelkeys.BookFileID, id))
	return d.execAffected(ctx, `DELETE FROM book_files WHERE id = $1`, id)
}

// GetFilesForBooks returns book files grouped by book ID for the given book IDs.
func (d *DB) GetFilesForBooks(ctx context.Context, bookIDs []string) (map[string][]BookFile, error) {
	orderBy := d.dialectOrderBy("bf.file_name", "ASC")
	return batchFetchByBookID(ctx, d, bookIDs, "db: batch fetching files for books",
		func(inClause string) string {
			return `SELECT ` + bookFileColumnsWithPrefix("bf.") + ` FROM book_files bf WHERE bf.book_id IN (` + inClause + `) ` + orderBy
		},
		func(row interface{ Scan(...any) error }) (string, *BookFile, error) {
			bf, err := scanBookFile(row)
			if err != nil {
				return "", nil, err
			}
			return bf.BookID, bf, nil
		},
	)
}

// IncrementBookFileDownloadCount atomically increments the download_count for the
// given book file by 1. Returns sql.ErrNoRows if the file does not exist.
func (d *DB) IncrementBookFileDownloadCount(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: incrementing book file download count", slog.String(otelkeys.BookFileID, id))
	return d.execAffected(ctx, `UPDATE book_files SET download_count = download_count + 1 WHERE id = $1`, id)
}
