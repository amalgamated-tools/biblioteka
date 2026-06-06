package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Sentinel errors returned by reading list write operations.
var (
	ErrReadingListNameExists  = errors.New("reading list name already exists")
	ErrInvalidReadingListName = errors.New("invalid reading list name")
)

// NormalizeReadingListName normalizes a reading list name by trimming
// whitespace and collapsing internal runs to a single space.
func NormalizeReadingListName(name string) string { return normalizeName(name) }

// ReadingList represents a row in the reading_lists table.
type ReadingList struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	BookCount   int       `json:"book_count"`
	CreatedAt   Timestamp `json:"created_at"`
	UpdatedAt   Timestamp `json:"updated_at"`
}

const (
	readingListColumns  = `rl.id, rl.user_id, rl.name, rl.description, (SELECT COUNT(*) FROM reading_list_books WHERE reading_list_id = rl.id), rl.created_at, rl.updated_at`
	readingListBaseFrom = `FROM reading_lists rl`
)

// scanReadingList scans a reading list row into a ReadingList struct.
func scanReadingList(row interface{ Scan(...any) error }) (*ReadingList, error) {
	return scanRow(row, func(r *ReadingList) []any {
		return []any{&r.ID, &r.UserID, &r.Name, &r.Description, &r.BookCount, &r.CreatedAt, &r.UpdatedAt}
	})
}

// CreateReadingList inserts a new reading list owned by userID. The name is
// normalized before storage. Returns ErrReadingListNameExists if a list with
// an equivalent normalized name already exists for this user.
func (d *DB) CreateReadingList(ctx context.Context, userID, name string, description *string) (*ReadingList, error) {
	return namedEntityCreate(ctx, "reading list", name,
		NormalizeReadingListName, ErrInvalidReadingListName, ErrReadingListNameExists,
		func(ctx context.Context, n string) (*ReadingList, error) {
			return scanReadingList(d.QueryRowContext(ctx,
				`INSERT INTO reading_lists (user_id, name, description) VALUES ($1, $2, $3)
         RETURNING id, user_id, name, description, 0, created_at, updated_at`,
				userID, n, description,
			))
		},
	)
}

// GetReadingList retrieves a reading list by ID scoped to the given user.
// Returns sql.ErrNoRows if not found or not owned by userID.
func (d *DB) GetReadingList(ctx context.Context, id, userID string) (*ReadingList, error) {
	slog.DebugContext(ctx, "db: fetching reading list",
		slog.String(otelkeys.ReadingListID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanReadingList(d.QueryRowContext(ctx,
		`SELECT `+readingListColumns+` `+readingListBaseFrom+`
         WHERE rl.id = $1 AND rl.user_id = $2`,
		id, userID,
	))
}

// ListReadingLists returns all reading lists for a user ordered by name,
// including a book_count populated via a LEFT JOIN.
func (d *DB) ListReadingLists(ctx context.Context, userID string) ([]ReadingList, error) {
	slog.DebugContext(ctx, "db: listing reading lists", slog.String(otelkeys.UserID, userID))
	rows, err := d.QueryContext(ctx,
		`SELECT `+readingListColumns+` `+readingListBaseFrom+`
         WHERE rl.user_id = $1
         ORDER BY rl.name ASC, rl.id ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanReadingList)
}

// UpdateReadingList updates the name and description of the reading list
// identified by id, scoped to userID. The name is normalized before storage.
// Returns sql.ErrNoRows if not found or not owned by userID.
// Returns ErrReadingListNameExists if the new name conflicts with another list
// owned by the same user.
func (d *DB) UpdateReadingList(ctx context.Context, id, userID, name string, description *string) (*ReadingList, error) {
	return namedEntityUpdate(ctx, "reading list", id, name,
		NormalizeReadingListName, ErrInvalidReadingListName, ErrReadingListNameExists,
		func(ctx context.Context, id, n string) (*ReadingList, error) {
			return scanReadingList(d.QueryRowContext(ctx,
				`UPDATE reading_lists SET name = $1, description = $2, updated_at = `+d.now()+`
         WHERE id = $3 AND user_id = $4
         RETURNING id, user_id, name, description,
           (SELECT COUNT(*) FROM reading_list_books WHERE reading_list_id = reading_lists.id),
           created_at, updated_at`,
				n, description, id, userID,
			))
		},
	)
}

// DeleteReadingList removes the reading list with the given ID, scoped to
// userID. Returns sql.ErrNoRows if not found or not owned by userID.
func (d *DB) DeleteReadingList(ctx context.Context, id, userID string) error {
	slog.DebugContext(ctx, "db: deleting reading list",
		slog.String(otelkeys.ReadingListID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return d.execAffected(ctx, `DELETE FROM reading_lists WHERE id = $1 AND user_id = $2`, id, userID)
}

// verifyReadingListOwnership checks that the reading list identified by listID
// is owned by userID. Returns the underlying error if the query fails (e.g.
// sql.ErrNoRows when the list does not exist), or sql.ErrNoRows if the list
// exists but belongs to a different user.
func (d *DB) verifyReadingListOwnership(ctx context.Context, listID, userID string) error {
	var ownerID string
	err := d.QueryRowContext(ctx,
		`SELECT user_id FROM reading_lists WHERE id = $1`,
		listID,
	).Scan(&ownerID)
	if err != nil {
		return err
	}
	if ownerID != userID {
		return sql.ErrNoRows
	}
	return nil
}

// AddBookToReadingList adds a book to a reading list. The list must be owned
// by userID. Returns ErrBookNotFound if the book does not exist.
// Returns (true, nil) if the book was newly added, (false, nil) if it was
// already present (idempotent).
func (d *DB) AddBookToReadingList(ctx context.Context, listID, userID, bookID string) (bool, error) {
	slog.DebugContext(ctx, "db: adding book to reading list",
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.BookID, bookID),
	)
	if err := d.verifyReadingListOwnership(ctx, listID, userID); err != nil {
		return false, err
	}
	result, err := d.ExecContext(ctx,
		`INSERT INTO reading_list_books (reading_list_id, book_id) VALUES ($1, $2)
         ON CONFLICT (reading_list_id, book_id) DO NOTHING`,
		listID, bookID,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			// Disambiguate: the violation could come from the book FK or from the
			// list FK (e.g. if the list was deleted between the ownership check and
			// the insert). Check whether the book actually exists to decide.
			var bookExists bool
			if scanErr := d.QueryRowContext(ctx,
				`SELECT EXISTS(SELECT 1 FROM books WHERE id = $1)`, bookID,
			).Scan(&bookExists); scanErr != nil || !bookExists {
				return false, fmt.Errorf("add book to reading list: %w", ErrBookNotFound)
			}
			// Book exists; the list must have been concurrently deleted.
			return false, sql.ErrNoRows
		}
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// RemoveBookFromReadingList removes a book from a reading list. The list must
// be owned by userID. Returns sql.ErrNoRows if the list doesn't exist or is
// not owned by the user. Returns (true, nil) if the book was removed,
// (false, nil) if it was not present.
func (d *DB) RemoveBookFromReadingList(ctx context.Context, listID, userID, bookID string) (bool, error) {
	slog.DebugContext(ctx, "db: removing book from reading list",
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.BookID, bookID),
	)
	if err := d.verifyReadingListOwnership(ctx, listID, userID); err != nil {
		return false, err
	}
	result, err := d.ExecContext(ctx,
		`DELETE FROM reading_list_books WHERE reading_list_id = $1 AND book_id = $2`,
		listID, bookID,
	)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListReadingListBooks returns a paginated list of books in a reading list,
// ordered by added_at. The list must be owned by userID.
// Returns sql.ErrNoRows if the list doesn't exist or is not owned by the user.
func (d *DB) ListReadingListBooks(ctx context.Context, listID, userID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing reading list books",
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)
	if err := d.verifyReadingListOwnership(ctx, listID, userID); err != nil {
		return nil, 0, err
	}

	return d.execBooksPaginated(
		ctx,
		offset,
		`SELECT `+bookColumns+`, COUNT(*) OVER() AS total
         FROM books b
         INNER JOIN reading_list_books rlb ON rlb.book_id = b.id
         WHERE rlb.reading_list_id = $1
         ORDER BY rlb.added_at ASC, b.id ASC
         LIMIT $2 OFFSET $3`,
		`SELECT COUNT(*) FROM reading_list_books WHERE reading_list_id = $1`,
		[]any{listID, limit, offset},
		[]any{listID},
	)
}

// GetReadingListsForBook returns all reading lists owned by userID that contain
// the given book, ordered by name.
func (d *DB) GetReadingListsForBook(ctx context.Context, bookID, userID string) ([]ReadingList, error) {
	slog.DebugContext(ctx, "db: getting reading lists for book",
		slog.String(otelkeys.BookID, bookID),
		slog.String(otelkeys.UserID, userID),
	)
	rows, err := d.QueryContext(ctx,
		`SELECT rl.id, rl.user_id, rl.name, rl.description, COUNT(rlb.book_id), rl.created_at, rl.updated_at
         FROM reading_lists rl
         LEFT JOIN reading_list_books rlb ON rlb.reading_list_id = rl.id
         WHERE rl.user_id = $1
           AND rl.id IN (SELECT reading_list_id FROM reading_list_books WHERE book_id = $2)
         GROUP BY rl.id, rl.user_id, rl.name, rl.description, rl.created_at, rl.updated_at
         ORDER BY rl.name ASC, rl.id ASC`,
		userID, bookID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanReadingList)
}
