package db

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// GoodreadsMetadata status constants.
const (
	GoodreadsMetadataStatusPending  = "pending"
	GoodreadsMetadataStatusApplied  = "applied"
	GoodreadsMetadataStatusRejected = "rejected"
)

// GoodreadsMetadata represents a row in the goodreads_metadata table.
type GoodreadsMetadata struct {
	ID                      string    `json:"id"`
	UserID                  string    `json:"user_id"`
	BookID                  *string   `json:"book_id"`
	Status                  string    `json:"status"`
	Title                   *string   `json:"title"`
	Description             *string   `json:"description"`
	ASIN                    *string   `json:"asin"`
	ISBN10                  *string   `json:"isbn10"`
	ISBN13                  *string   `json:"isbn13"`
	GoodreadsID             *string   `json:"goodreads_id"`
	HardcoverID             *string   `json:"hardcover_id"`
	GoogleBooksID           *string   `json:"google_books_id"`
	PublicationDate         *string   `json:"publication_date"`
	Publisher               *string   `json:"publisher"`
	Language                *string   `json:"language"`
	CoverImageURL           *string   `json:"cover_image_url"`
	AuthorName              *string   `json:"author_name"`
	AuthorGoodreadsID       *string   `json:"author_goodreads_id"`
	AuthorImageURL          *string   `json:"author_image_url"`
	GoodreadsWorkID         *string   `json:"goodreads_work_id"`
	GoodreadsBookLegacyID   *int64    `json:"goodreads_book_legacy_id"`
	GoodreadsWorkLegacyID   *int64    `json:"goodreads_work_legacy_id"`
	GoodreadsAuthorLegacyID *int64    `json:"goodreads_author_legacy_id"`
	CreatedAt               Timestamp `json:"created_at"`
	UpdatedAt               Timestamp `json:"updated_at"`
}

const goodreadsMetadataColumns = `id, user_id, book_id, status, title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, cover_image_url, author_name, author_goodreads_id, author_image_url, goodreads_work_id, goodreads_book_legacy_id, goodreads_work_legacy_id, goodreads_author_legacy_id, created_at, updated_at`

func scanGoodreadsMetadata(row interface{ Scan(...any) error }) (*GoodreadsMetadata, error) {
	var gm GoodreadsMetadata
	err := row.Scan(
		&gm.ID, &gm.UserID, &gm.BookID, &gm.Status,
		&gm.Title, &gm.Description, &gm.ASIN, &gm.ISBN10, &gm.ISBN13,
		&gm.GoodreadsID, &gm.HardcoverID, &gm.GoogleBooksID,
		&gm.PublicationDate, &gm.Publisher, &gm.Language,
		&gm.CoverImageURL, &gm.AuthorName, &gm.AuthorGoodreadsID, &gm.AuthorImageURL,
		&gm.GoodreadsWorkID, &gm.GoodreadsBookLegacyID, &gm.GoodreadsWorkLegacyID,
		&gm.GoodreadsAuthorLegacyID, &gm.CreatedAt, &gm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &gm, nil
}

// CreateGoodreadsMetadata inserts a new goodreads_metadata row and returns it.
func (d *DB) CreateGoodreadsMetadata(ctx context.Context, userID string, bookID, title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, coverImageURL, authorName, authorGoodreadsID, authorImageURL, goodreadsWorkID *string, goodreadsBookLegacyID, goodreadsWorkLegacyID, goodreadsAuthorLegacyID *int64) (*GoodreadsMetadata, error) {
	slog.DebugContext(ctx, "db: creating goodreads metadata", slog.String(otelkeys.UserID, userID))
	return scanGoodreadsMetadata(d.QueryRowContext(ctx,
		`INSERT INTO goodreads_metadata (user_id, book_id, title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, cover_image_url, author_name, author_goodreads_id, author_image_url, goodreads_work_id, goodreads_book_legacy_id, goodreads_work_legacy_id, goodreads_author_legacy_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21) RETURNING `+goodreadsMetadataColumns,
		userID, bookID, title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, coverImageURL, authorName, authorGoodreadsID, authorImageURL, goodreadsWorkID, goodreadsBookLegacyID, goodreadsWorkLegacyID, goodreadsAuthorLegacyID,
	))
}

// GetGoodreadsMetadata returns a goodreads_metadata row by ID for the given user.
func (d *DB) GetGoodreadsMetadata(ctx context.Context, userID, id string) (*GoodreadsMetadata, error) {
	slog.DebugContext(ctx, "db: fetching goodreads metadata",
		slog.String(otelkeys.GoodreadsMetadataID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanGoodreadsMetadata(d.QueryRowContext(ctx,
		`SELECT `+goodreadsMetadataColumns+` FROM goodreads_metadata WHERE id = $1 AND user_id = $2`,
		id, userID,
	))
}

// ListGoodreadsMetadataByUser returns all goodreads_metadata rows for a user, ordered by created_at DESC.
func (d *DB) ListGoodreadsMetadataByUser(ctx context.Context, userID string, limit, offset int) ([]GoodreadsMetadata, error) {
	slog.DebugContext(ctx, "db: listing goodreads metadata by user",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)
	rows, err := d.QueryContext(ctx,
		`SELECT `+goodreadsMetadataColumns+` FROM goodreads_metadata WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []GoodreadsMetadata
	for rows.Next() {
		gm, err := scanGoodreadsMetadata(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *gm)
	}
	return results, rows.Err()
}

// ListGoodreadsMetadataByStatus returns goodreads_metadata rows for a user filtered by status.
func (d *DB) ListGoodreadsMetadataByStatus(ctx context.Context, userID, status string, limit, offset int) ([]GoodreadsMetadata, error) {
	slog.DebugContext(ctx, "db: listing goodreads metadata by status",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Status, status),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)
	rows, err := d.QueryContext(ctx,
		`SELECT `+goodreadsMetadataColumns+` FROM goodreads_metadata WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, status, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []GoodreadsMetadata
	for rows.Next() {
		gm, err := scanGoodreadsMetadata(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *gm)
	}
	return results, rows.Err()
}

// UpdateGoodreadsMetadataStatus updates the status of a goodreads_metadata row for the given user.
func (d *DB) UpdateGoodreadsMetadataStatus(ctx context.Context, userID, id, status string) (*GoodreadsMetadata, error) {
	slog.DebugContext(ctx, "db: updating goodreads metadata status",
		slog.String(otelkeys.GoodreadsMetadataID, id),
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Status, status),
	)
	return scanGoodreadsMetadata(d.QueryRowContext(ctx,
		`UPDATE goodreads_metadata SET status = $1, updated_at = `+d.now()+` WHERE id = $2 AND user_id = $3 RETURNING `+goodreadsMetadataColumns,
		status, id, userID,
	))
}

// DeleteGoodreadsMetadata deletes a goodreads_metadata row by ID for the given user.
func (d *DB) DeleteGoodreadsMetadata(ctx context.Context, userID, id string) error {
	slog.DebugContext(ctx, "db: deleting goodreads metadata",
		slog.String(otelkeys.GoodreadsMetadataID, id),
		slog.String(otelkeys.UserID, userID),
	)
	res, err := d.ExecContext(ctx,
		`DELETE FROM goodreads_metadata WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
