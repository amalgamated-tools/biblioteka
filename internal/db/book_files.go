package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookFile represents a row in the book_files table.
type BookFile struct {
	ID        string    `json:"id"`
	BookID    string    `json:"book_id"`
	FileType  string    `json:"file_type"`
	FileName  string    `json:"file_name"`
	FileSize  int64     `json:"file_size"`
	FileHash  *string   `json:"file_hash"`
	FilePath  string    `json:"file_path"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

const bookFileColumns = `id, book_id, file_type, file_name, file_size, file_hash, file_path, created_at, updated_at`

func scanBookFile(row interface{ Scan(...any) error }) (*BookFile, error) {
	var bf BookFile
	err := row.Scan(&bf.ID, &bf.BookID, &bf.FileType, &bf.FileName, &bf.FileSize, &bf.FileHash, &bf.FilePath, &bf.CreatedAt, &bf.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &bf, nil
}

// bookFileColumnsWithPrefix returns book_files columns with a table alias prefix.
func bookFileColumnsWithPrefix(prefix string) string {
	return prefix + "id, " + prefix + "book_id, " + prefix + "file_type, " + prefix + "file_name, " + prefix + "file_size, " + prefix + "file_hash, " + prefix + "file_path, " + prefix + "created_at, " + prefix + "updated_at"
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
	slog.DebugContext(ctx, "db: fetching book file", slog.String(otelkeys.ID, id))
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
	defer rows.Close()

	var files []BookFile
	for rows.Next() {
		bf, err := scanBookFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, *bf)
	}
	return files, rows.Err()
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
	slog.DebugContext(ctx, "db: deleting book file", slog.String(otelkeys.ID, id))
	res, err := d.ExecContext(ctx, `DELETE FROM book_files WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetFilesForBooks returns book files grouped by book ID for the given book IDs.
func (d *DB) GetFilesForBooks(ctx context.Context, bookIDs []string) (map[string][]BookFile, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, "db: batch fetching files for books", slog.Int(otelkeys.BookCount, len(bookIDs)))

	placeholders := make([]string, len(bookIDs))
	args := make([]any, len(bookIDs))
	for i, id := range bookIDs {
		placeholders[i] = dollarN(i + 1)
		args[i] = id
	}

	orderBy := d.dialectOrderBy("bf.file_name", "ASC")
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookFileColumnsWithPrefix("bf.")+` FROM book_files bf WHERE bf.book_id IN (`+strings.Join(placeholders, ",")+`) `+orderBy,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]BookFile, len(bookIDs))
	for rows.Next() {
		bf, err := scanBookFile(rows)
		if err != nil {
			return nil, err
		}
		result[bf.BookID] = append(result[bf.BookID], *bf)
	}
	return result, rows.Err()
}
