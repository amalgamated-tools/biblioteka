package db

import (
	"context"
	"database/sql"
	"log/slog"

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
	orderBy := "ORDER BY file_name ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY file_name ASC, id ASC"
	}
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
