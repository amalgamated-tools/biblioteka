package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// BookAnnotation represents a user annotation on a book.
type BookAnnotation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	BookID    string    `json:"book_id"`
	Text      string    `json:"text"`
	CFI       *string   `json:"cfi,omitempty"`
	GroupID   *string   `json:"group_id,omitempty"`
	UserName  string    `json:"user_name"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

const bookAnnotationColumns = `ba.id, ba.user_id, ba.book_id, ba.text, ba.cfi, ba.group_id, u.name, ba.created_at, ba.updated_at`

func scanBookAnnotation(row interface{ Scan(...any) error }) (*BookAnnotation, error) {
	return scanRow(row, func(a *BookAnnotation) []any {
		return []any{&a.ID, &a.UserID, &a.BookID, &a.Text, &a.CFI, &a.GroupID, &a.UserName, &a.CreatedAt, &a.UpdatedAt}
	})
}

// CreateAnnotation creates a new book annotation.
func (d *DB) CreateAnnotation(ctx context.Context, userID, bookID, text string, cfi, groupID *string) (*BookAnnotation, error) {
	slog.DebugContext(ctx, "db: creating book annotation",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.BookID, bookID),
	)
	return scanBookAnnotation(d.QueryRowContext(ctx,
		`INSERT INTO book_annotations (user_id, book_id, text, cfi, group_id) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, book_id, text, cfi, group_id,
		   (SELECT name FROM users WHERE id = $1),
		   created_at, updated_at`,
		userID, bookID, text, cfi, groupID,
	))
}

// GetAnnotation retrieves an annotation by ID scoped to the owning user.
func (d *DB) GetAnnotation(ctx context.Context, id, userID string) (*BookAnnotation, error) {
	slog.DebugContext(ctx, "db: fetching book annotation",
		slog.String(otelkeys.AnnotationID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanBookAnnotation(d.QueryRowContext(ctx,
		`SELECT `+bookAnnotationColumns+`
		 FROM book_annotations ba
		 JOIN users u ON u.id = ba.user_id
		 WHERE ba.id = $1 AND ba.user_id = $2`,
		id, userID,
	))
}

// ListAnnotationsForBook returns annotations for a book visible to the user:
// their own annotations and annotations shared with groups they belong to.
func (d *DB) ListAnnotationsForBook(ctx context.Context, bookID, userID string) ([]BookAnnotation, error) {
	slog.DebugContext(ctx, "db: listing book annotations",
		slog.String(otelkeys.BookID, bookID),
		slog.String(otelkeys.UserID, userID),
	)
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookAnnotationColumns+`
		 FROM book_annotations ba
		 JOIN users u ON u.id = ba.user_id
		 WHERE ba.book_id = $1
		   AND (
		     ba.user_id = $2
		     OR (ba.group_id IS NOT NULL AND ba.group_id IN (
		       SELECT group_id FROM reading_group_members WHERE user_id = $2
		     ))
		   )
		 ORDER BY ba.created_at ASC`,
		bookID, userID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBookAnnotation)
}

// UpdateAnnotation updates an annotation's text, cfi, and group_id. Only the owning user can update.
func (d *DB) UpdateAnnotation(ctx context.Context, id, userID, text string, cfi, groupID *string) (*BookAnnotation, error) {
	slog.DebugContext(ctx, "db: updating book annotation",
		slog.String(otelkeys.AnnotationID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanBookAnnotation(d.QueryRowContext(ctx,
		`UPDATE book_annotations SET text = $1, cfi = $2, group_id = $3, updated_at = `+d.now()+`
		 WHERE id = $4 AND user_id = $5
		 RETURNING id, user_id, book_id, text, cfi, group_id,
		   (SELECT name FROM users WHERE id = book_annotations.user_id),
		   created_at, updated_at`,
		text, cfi, groupID, id, userID,
	))
}

// DeleteAnnotation deletes an annotation. Only the owning user can delete.
func (d *DB) DeleteAnnotation(ctx context.Context, id, userID string) error {
	slog.DebugContext(ctx, "db: deleting book annotation",
		slog.String(otelkeys.AnnotationID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return d.execAffected(ctx,
		`DELETE FROM book_annotations WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
}
