package db

import "database/sql"

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
func (d *DB) CreateBookFile(bookID, fileType, fileName string, fileSize int64, fileHash *string, filePath string) (*BookFile, error) {
	bf, err := scanBookFile(d.QueryRow(
		`INSERT INTO book_files (book_id, file_type, file_name, file_size, file_hash, file_path) VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+bookFileColumns,
		bookID, fileType, fileName, fileSize, fileHash, filePath,
	))
	if err != nil {
		return nil, err
	}
	return bf, nil
}

// GetBookFile returns a book file by ID, or sql.ErrNoRows if not found.
func (d *DB) GetBookFile(id string) (*BookFile, error) {
	return scanBookFile(d.QueryRow(
		`SELECT `+bookFileColumns+` FROM book_files WHERE id = $1`,
		id,
	))
}

// ListBookFiles returns all files for a given book.
func (d *DB) ListBookFiles(bookID string) ([]BookFile, error) {
	orderBy := "ORDER BY file_name ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY file_name ASC, id ASC"
	}
	rows, err := d.Query(
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
func (d *DB) DeleteBookFile(id string) error {
	res, err := d.Exec(`DELETE FROM book_files WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
