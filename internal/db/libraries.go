package db

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"
)

// ErrLibraryNameExists is returned when a library with the given name already exists.
var ErrLibraryNameExists = errors.New("library name already exists")

// Library represents a row in the libraries table.
type Library struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Paths            string    `json:"paths"`
	OrganizationType string    `json:"organization_type"`
	Monitored        bool      `json:"monitored"`
	CreatedAt        Timestamp `json:"created_at"`
	UpdatedAt        Timestamp `json:"updated_at"`
}

const libraryColumns = `id, name, paths, organization_type, monitored, created_at, updated_at`

// scanLibrary scans a library row into a Library struct.
func scanLibrary(row interface{ Scan(...any) error }) (*Library, error) {
	var lib Library
	err := row.Scan(&lib.ID, &lib.Name, &lib.Paths, &lib.OrganizationType, &lib.Monitored, &lib.CreatedAt, &lib.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

// CreateLibrary inserts a new library and returns it.
// Returns ErrLibraryNameExists if a library with that name already exists.
func (d *DB) CreateLibrary(name, paths, organizationType string, monitored bool) (*Library, error) {
	slog.Debug("db: creating library", slog.String("name", name))
	lib, err := scanLibrary(d.QueryRow(
		`INSERT INTO libraries (name, paths, organization_type, monitored) VALUES ($1, $2, $3, $4) RETURNING `+libraryColumns,
		name, paths, organizationType, monitored,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrLibraryNameExists
		}
		return nil, err
	}
	return lib, nil
}

// GetLibrary returns a library by ID, or sql.ErrNoRows if not found.
func (d *DB) GetLibrary(id string) (*Library, error) {
	slog.Debug("db: fetching library", slog.String("id", id))
	return scanLibrary(d.QueryRow(
		`SELECT `+libraryColumns+` FROM libraries WHERE id = $1`,
		id,
	))
}

// ListLibraries returns all libraries ordered by creation time.
func (d *DB) ListLibraries() ([]Library, error) {
	slog.Debug("db: listing libraries")
	orderBy := "ORDER BY created_at ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY created_at ASC, id ASC"
	}
	rows, err := d.Query(
		`SELECT ` + libraryColumns + ` FROM libraries ` + orderBy,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []Library
	for rows.Next() {
		lib, err := scanLibrary(rows)
		if err != nil {
			return nil, err
		}
		libraries = append(libraries, *lib)
	}
	return libraries, rows.Err()
}

// UpdateLibrary updates a library's fields and returns the updated library.
// Returns sql.ErrNoRows if the library doesn't exist.
// Returns ErrLibraryNameExists if the new name conflicts with another library.
func (d *DB) UpdateLibrary(id, name, paths, organizationType string, monitored bool) (*Library, error) {
	slog.Debug("db: updating library", slog.String("id", id), slog.String("name", name))
	lib, err := scanLibrary(d.QueryRow(
		`UPDATE libraries SET name = $1, paths = $2, organization_type = $3, monitored = $4, updated_at = `+d.now()+` WHERE id = $5 RETURNING `+libraryColumns,
		name, paths, organizationType, monitored, id,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrLibraryNameExists
		}
		return nil, err
	}
	return lib, nil
}

// DeleteLibrary removes a library by ID.
// Returns sql.ErrNoRows if the library doesn't exist.
func (d *DB) DeleteLibrary(id string) error {
	slog.Debug("db: deleting library", slog.String("id", id))
	res, err := d.Exec(`DELETE FROM libraries WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// isUniqueViolation checks if an error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	msg := err.Error()
	// SQLite: "UNIQUE constraint failed: ..."
	// PostgreSQL: "duplicate key value violates unique constraint ..."
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}
