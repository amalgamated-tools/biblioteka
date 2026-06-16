package db

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Sentinel errors returned by tag write operations.
var (
	ErrTagNameExists  = errors.New("tag name already exists")
	ErrInvalidTagName = errors.New("invalid tag name")
)

// NormalizeTagName normalizes a tag name by trimming whitespace and
// collapsing internal runs to a single space while preserving capitalization.
func NormalizeTagName(name string) string { return normalizeName(name) }

// Tag represents a row in the tags table.
type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

const tagColumns = `id, name, created_at, updated_at`

type tagListQuery struct{}

func (tagListQuery) table() string   { return "tags" }
func (tagListQuery) columns() string { return tagColumns }
func (tagListQuery) orderBy(d *DB) string {
	return d.dialectOrderBy("name", "ASC")
}

func scanTag(row interface{ Scan(...any) error }) (*Tag, error) {
	return scanRow(row, func(t *Tag) []any {
		return []any{&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt}
	})
}

// CreateTag inserts a new tag with the given name. The name is normalized
// before storage. Returns ErrTagNameExists if a tag with an equivalent
// normalized name already exists.
func (d *DB) CreateTag(ctx context.Context, name string) (*Tag, error) {
	return namedEntityCreate(ctx, "tag", name, NormalizeTagName, ErrInvalidTagName, ErrTagNameExists,
		func(ctx context.Context, n string) (*Tag, error) {
			return scanTag(d.QueryRowContext(ctx,
				`INSERT INTO tags (name) VALUES ($1) RETURNING `+tagColumns,
				n,
			))
		},
	)
}

// GetTag retrieves a tag by its UUID. Returns sql.ErrNoRows if not found.
func (d *DB) GetTag(ctx context.Context, id string) (*Tag, error) {
	slog.DebugContext(ctx, "db: fetching tag", slog.String(otelkeys.TagID, id))
	return scanTag(d.QueryRowContext(ctx,
		`SELECT `+tagColumns+` FROM tags WHERE id = $1`,
		id,
	))
}

// GetTagByName looks up a tag by name using case-insensitive matching after
// normalizing whitespace.
func (d *DB) GetTagByName(ctx context.Context, name string) (*Tag, error) {
	name = NormalizeTagName(name)
	slog.DebugContext(ctx, "db: fetching tag by name", slog.String(otelkeys.Name, name))
	return scanTag(d.QueryRowContext(ctx,
		`SELECT `+tagColumns+` FROM tags WHERE LOWER(name) = LOWER($1)`,
		name,
	))
}

// ListTags returns all tags ordered by name.
func (d *DB) ListTags(ctx context.Context) ([]Tag, error) {
	slog.DebugContext(ctx, "db: listing tags")
	return listAll(ctx, d, tagListQuery{}, scanTag)
}

// ListTagsPaginated returns tags ordered by name with pagination and total count.
func (d *DB) ListTagsPaginated(ctx context.Context, limit, offset int) ([]Tag, int, error) {
	slog.DebugContext(ctx, "db: listing tags paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)
	return listPaginated(ctx, d, tagListQuery{}, limit, offset, scanTag)
}

// UpdateTag replaces the name of the tag identified by id. The name is
// normalized. Returns sql.ErrNoRows if the tag does not exist, or
// ErrTagNameExists if the new name conflicts with another tag.
func (d *DB) UpdateTag(ctx context.Context, id, name string) (*Tag, error) {
	return namedEntityUpdate(ctx, "tag", id, name, NormalizeTagName, ErrInvalidTagName, ErrTagNameExists,
		func(ctx context.Context, entityID, n string) (*Tag, error) {
			return scanTag(d.QueryRowContext(ctx,
				`UPDATE tags SET name = $1, updated_at = `+d.now()+` WHERE id = $2 RETURNING `+tagColumns,
				n, entityID,
			))
		},
	)
}

// FindOrCreateTag looks up a tag by name (case-insensitive) and returns it,
// creating a new one if it doesn't exist. Handles concurrent insert races
// gracefully.
func (d *DB) FindOrCreateTag(ctx context.Context, name string) (*Tag, error) {
	return findOrCreate(ctx, name, "tag",
		NormalizeTagName, ErrInvalidTagName, ErrTagNameExists,
		d.GetTagByName,
		func(ctx context.Context, n string) (*Tag, error) {
			return d.CreateTag(ctx, n)
		},
	)
}

// DeleteTag removes the tag with the given ID. Returns sql.ErrNoRows if no
// matching tag exists.
func (d *DB) DeleteTag(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting tag", slog.String(otelkeys.TagID, id))
	return d.execAffected(ctx, `DELETE FROM tags WHERE id = $1`, id)
}

// GetBookTags returns all tags for a book, ordered by name.
func (d *DB) GetBookTags(ctx context.Context, bookID string) ([]Tag, error) {
	slog.DebugContext(ctx, "db: fetching book tags", slog.String(otelkeys.BookID, bookID))
	rows, err := d.QueryContext(ctx,
		`SELECT t.id, t.name, t.created_at, t.updated_at FROM tags t INNER JOIN book_tags bt ON bt.tag_id = t.id WHERE bt.book_id = $1 ORDER BY LOWER(t.name) ASC`,
		bookID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanTag)
}

// GetTagsForBooks returns tags grouped by book ID for the given book IDs.
func (d *DB) GetTagsForBooks(ctx context.Context, bookIDs []string) (map[string][]Tag, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, "db: batch fetching tags for books", slog.Int(otelkeys.BookCount, len(bookIDs)))

	inClause, args := buildInClause(bookIDs, 1)

	rows, err := d.QueryContext(ctx,
		`SELECT bt.book_id, t.id, t.name, t.created_at, t.updated_at
		FROM tags t INNER JOIN book_tags bt ON bt.tag_id = t.id
		WHERE bt.book_id IN (`+inClause+`)
		ORDER BY LOWER(t.name) ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}

	return collectRowsGrouped(rows, func(row interface{ Scan(...any) error }) (string, *Tag, error) {
		var bookID string
		t, err := scanTag(prefixedScanner{row: row, prefix: []any{&bookID}})
		if err != nil {
			return "", nil, err
		}
		return bookID, t, nil
	}, len(bookIDs))
}

// SetBookTags replaces all tag associations for a book atomically.
// Duplicate tag IDs are silently deduplicated.
func (d *DB) SetBookTags(ctx context.Context, bookID string, tagIDs []string) error {
	slog.DebugContext(ctx, "db: setting book tags",
		slog.String(otelkeys.BookID, bookID),
		slog.Int(otelkeys.Count, len(tagIDs)),
	)
	return d.replaceBookAssociations(ctx, bookID, tagIDs,
		`DELETE FROM book_tags WHERE book_id = $1`,
		`INSERT INTO book_tags (book_id, tag_id) VALUES ($1, $2)`,
	)
}
