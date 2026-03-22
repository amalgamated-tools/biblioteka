package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ErrLibraryNameExists is returned when a library with the given name already exists.
var ErrLibraryNameExists = errors.New("library name already exists")

const (
	LibraryOrganizationBookPerFolder = "book_per_folder"
	LibraryOrganizationBookPerFile   = "book_per_file"
	LibraryOrganizationNone          = "none"
)

var libraryOrganizationTypes = []string{
	LibraryOrganizationBookPerFolder,
	LibraryOrganizationBookPerFile,
	LibraryOrganizationNone,
}

// LibraryOrganizationTypeNames returns the supported library organization type
// values. The returned slice is a copy safe for the caller to modify.
func LibraryOrganizationTypeNames() []string {
	cpy := make([]string, len(libraryOrganizationTypes))
	copy(cpy, libraryOrganizationTypes)
	return cpy
}

func IsValidLibraryOrganizationType(organizationType string) bool {
	return slices.Contains(libraryOrganizationTypes, organizationType)
}

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
func scanLibrary(ctx context.Context, row interface{ Scan(...any) error }) (*Library, error) {
	var lib Library
	err := row.Scan(&lib.ID, &lib.Name, &lib.Paths, &lib.OrganizationType, &lib.Monitored, &lib.CreatedAt, &lib.UpdatedAt)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to scan library", slog.Any("error", err))
		return nil, fmt.Errorf("scanning library: %w", err)
	}
	return &lib, nil
}

// CreateLibrary inserts a new library and returns it.
// Returns ErrLibraryNameExists if a library with that name already exists.
func (d *DB) CreateLibrary(ctx context.Context, name, paths, organizationType string, monitored bool) (*Library, error) {
	slog.DebugContext(ctx, "db: creating library", slog.String(otelkeys.Name, name))
	lib, err := scanLibrary(ctx, d.QueryRowContext(ctx,
		`INSERT INTO libraries (name, paths, organization_type, monitored) VALUES ($1, $2, $3, $4) RETURNING `+libraryColumns,
		name, paths, organizationType, monitored,
	))
	if err != nil {
		if isUniqueViolation(err) {
			slog.ErrorContext(ctx, "Library name already exists", slog.String(otelkeys.Name, name))
			return nil, ErrLibraryNameExists
		}
		slog.ErrorContext(ctx, "Failed to create library", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create library: %w", err)
	}

	return lib, nil
}

// GetLibrary returns a library by ID, or sql.ErrNoRows if not found.
func (d *DB) GetLibrary(ctx context.Context, id string) (*Library, error) {
	slog.DebugContext(ctx, "db: fetching library", slog.String(otelkeys.ID, id))
	return scanLibrary(ctx, d.QueryRowContext(ctx,
		`SELECT `+libraryColumns+` FROM libraries WHERE id = $1`,
		id,
	))
}

// ListLibraries returns all libraries ordered by creation time.
func (d *DB) ListLibraries(ctx context.Context) ([]Library, error) {
	slog.DebugContext(ctx, "db: listing libraries")
	orderBy := "ORDER BY created_at ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY created_at ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+libraryColumns+` FROM libraries `+orderBy,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to query libraries", slog.Any("error", err))
		return nil, fmt.Errorf("failed to query libraries: %w", err)
	}
	defer rows.Close()

	var libraries []Library
	for rows.Next() {
		lib, err := scanLibrary(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan library row", slog.Any("error", err))
			return nil, fmt.Errorf("failed to scan library row: %w", err)
		}
		libraries = append(libraries, *lib)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Error iterating library rows", slog.Any("error", err))
		return nil, fmt.Errorf("error iterating library rows: %w", err)
	}
	return libraries, nil
}

// UpdateLibrary updates a library's fields and returns the updated library.
// Returns sql.ErrNoRows if the library doesn't exist.
// Returns ErrLibraryNameExists if the new name conflicts with another library.
func (d *DB) UpdateLibrary(ctx context.Context, id, name, paths, organizationType string, monitored bool) (*Library, error) {
	slog.DebugContext(ctx, "db: updating library",
		slog.String(otelkeys.ID, id),
		slog.String(otelkeys.Name, name),
	)
	lib, err := scanLibrary(ctx, d.QueryRowContext(ctx,
		`UPDATE libraries SET name = $1, paths = $2, organization_type = $3, monitored = $4, updated_at = `+d.now()+` WHERE id = $5 RETURNING `+libraryColumns,
		name, paths, organizationType, monitored, id,
	))
	if err != nil {
		if isUniqueViolation(err) {
			slog.ErrorContext(ctx, "Library name already exists", slog.String(otelkeys.Name, name))
			return nil, ErrLibraryNameExists
		}
		slog.ErrorContext(ctx, "Failed to update library", slog.Any("error", err))
		return nil, fmt.Errorf("failed to update library: %w", err)
	}
	return lib, nil
}

// DeleteLibrary removes a library by ID.
// Returns sql.ErrNoRows if the library doesn't exist.
func (d *DB) DeleteLibrary(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting library", slog.String(otelkeys.ID, id))
	res, err := d.ExecContext(ctx, `DELETE FROM libraries WHERE id = $1`, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to delete library", slog.Any("error", err))
		return fmt.Errorf("failed to delete library: %w", err)
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
