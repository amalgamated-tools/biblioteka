package db

import (
	"database/sql"
	"errors"
	"strings"
)

// ErrLibraryNameExists is returned when a library with the given name already exists for the user.
var ErrLibraryNameExists = errors.New("library name already exists")

// Library represents a row in the libraries table.
type Library struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Name             string    `json:"name"`
	Paths            string    `json:"paths"`
	OrganizationType string    `json:"organization_type"`
	Monitored        bool      `json:"monitored"`
	CreatedAt        Timestamp `json:"created_at"`
	UpdatedAt        Timestamp `json:"updated_at"`
}

const libraryColumns = `id, user_id, name, paths, organization_type, monitored, created_at, updated_at`

// scanLibrary scans a library row into a Library struct.
func scanLibrary(row interface{ Scan(...any) error }) (*Library, error) {
	var lib Library
	err := row.Scan(&lib.ID, &lib.UserID, &lib.Name, &lib.Paths, &lib.OrganizationType, &lib.Monitored, &lib.CreatedAt, &lib.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

// CreateLibrary inserts a new library for the given user and returns it.
// Returns ErrLibraryNameExists if the user already has a library with that name.
func (d *DB) CreateLibrary(userID, name, paths, organizationType string, monitored bool) (*Library, error) {
	lib, err := scanLibrary(d.QueryRow(
		`INSERT INTO libraries (user_id, name, paths, organization_type, monitored) VALUES ($1, $2, $3, $4, $5) RETURNING `+libraryColumns,
		userID, name, paths, organizationType, monitored,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrLibraryNameExists
		}
		return nil, err
	}
	return lib, nil
}

// GetLibrary returns a library by ID for the given user, or sql.ErrNoRows if not found.
func (d *DB) GetLibrary(userID, id string) (*Library, error) {
	return scanLibrary(d.QueryRow(
		`SELECT `+libraryColumns+` FROM libraries WHERE id = $1 AND user_id = $2`,
		id, userID,
	))
}

// ListLibraries returns all libraries for the given user ordered by creation time.
func (d *DB) ListLibraries(userID string) ([]Library, error) {
	orderBy := "ORDER BY created_at ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY created_at ASC, id ASC"
	}
	rows, err := d.Query(
		`SELECT `+libraryColumns+` FROM libraries WHERE user_id = $1 `+orderBy,
		userID,
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
// Returns sql.ErrNoRows if the library doesn't exist for the given user.
// Returns ErrLibraryNameExists if the new name conflicts with another library.
func (d *DB) UpdateLibrary(userID, id, name, paths, organizationType string, monitored bool) (*Library, error) {
	lib, err := scanLibrary(d.QueryRow(
		`UPDATE libraries SET name = $1, paths = $2, organization_type = $3, monitored = $4, updated_at = `+d.now()+` WHERE id = $5 AND user_id = $6 RETURNING `+libraryColumns,
		name, paths, organizationType, monitored, id, userID,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrLibraryNameExists
		}
		return nil, err
	}
	return lib, nil
}

// DeleteLibrary removes a library by ID for the given user.
// Returns sql.ErrNoRows if the library doesn't exist for the given user.
func (d *DB) DeleteLibrary(userID, id string) error {
	res, err := d.Exec(`DELETE FROM libraries WHERE id = $1 AND user_id = $2`, id, userID)
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
