package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// AI enrichment status constants.
const (
	AIEnrichmentStatusPending  = "pending"
	AIEnrichmentStatusApplied  = "applied"
	AIEnrichmentStatusRejected = "rejected"
)

// ErrInvalidAIEnrichmentStatus is returned when an invalid status is passed.
var ErrInvalidAIEnrichmentStatus = errors.New("db: invalid ai_enrichment status")

// ErrAIEnrichmentNotPending is returned when an enrichment is no longer in pending status.
var ErrAIEnrichmentNotPending = errors.New("db: ai_enrichment is not in pending status")

// AIEnrichment represents a row in the ai_enrichments table.
type AIEnrichment struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	BookID               *string   `json:"book_id"`
	Status               string    `json:"status"`
	Provider             string    `json:"provider"`
	Model                string    `json:"model"`
	SuggestedTags        []string  `json:"suggested_tags"`
	ReadingLevel         *string   `json:"reading_level"`
	GeneratedDescription *string   `json:"generated_description"`
	RawResponse          string    `json:"raw_response"`
	CreatedAt            Timestamp `json:"created_at"`
	UpdatedAt            Timestamp `json:"updated_at"`
}

const aiEnrichmentColumns = `id, user_id, book_id, status, provider, model, suggested_tags, reading_level, generated_description, raw_response, created_at, updated_at`

func scanAIEnrichment(row interface{ Scan(...any) error }) (*AIEnrichment, error) {
	var suggestedTagsJSON string
	e, err := scanRow(row, func(a *AIEnrichment) []any {
		return []any{
			&a.ID, &a.UserID, &a.BookID, &a.Status, &a.Provider, &a.Model,
			&suggestedTagsJSON, &a.ReadingLevel, &a.GeneratedDescription,
			&a.RawResponse, &a.CreatedAt, &a.UpdatedAt,
		}
	})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(suggestedTagsJSON), &e.SuggestedTags); err != nil {
		return nil, fmt.Errorf("unmarshal suggested_tags: %w", err)
	}
	return e, nil
}

// CreateAIEnrichment inserts a new AI enrichment record with pending status.
func (d *DB) CreateAIEnrichment(
	ctx context.Context,
	userID string,
	bookID *string,
	provider, model string,
	suggestedTags []string,
	readingLevel *string,
	generatedDescription *string,
	rawResponse string,
) (*AIEnrichment, error) {
	slog.DebugContext(ctx, "db: creating AI enrichment",
		slog.String(otelkeys.UserID, userID),
	)

	if suggestedTags == nil {
		suggestedTags = []string{}
	}
	tagsJSON, err := json.Marshal(suggestedTags)
	if err != nil {
		return nil, fmt.Errorf("marshal suggested_tags: %w", err)
	}

	return scanAIEnrichment(d.QueryRowContext(ctx,
		`INSERT INTO ai_enrichments (user_id, book_id, provider, model, suggested_tags, reading_level, generated_description, raw_response) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+aiEnrichmentColumns,
		userID, bookID, provider, model, string(tagsJSON), readingLevel, generatedDescription, rawResponse,
	))
}

// GetAIEnrichment retrieves an AI enrichment by ID for the given user.
// Returns sql.ErrNoRows if not found.
func (d *DB) GetAIEnrichment(ctx context.Context, userID, id string) (*AIEnrichment, error) {
	slog.DebugContext(ctx, "db: fetching AI enrichment",
		slog.String(otelkeys.AIEnrichmentID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanAIEnrichment(d.QueryRowContext(ctx,
		`SELECT `+aiEnrichmentColumns+` FROM ai_enrichments WHERE id = $1 AND user_id = $2`,
		id, userID,
	))
}

// GetPendingAIEnrichmentByBook returns the most recent pending AI enrichment
// for the given book and user, or sql.ErrNoRows if none exists.
func (d *DB) GetPendingAIEnrichmentByBook(ctx context.Context, userID, bookID string) (*AIEnrichment, error) {
	slog.DebugContext(ctx, "db: fetching pending AI enrichment by book",
		slog.String(otelkeys.BookID, bookID),
		slog.String(otelkeys.UserID, userID),
	)
	return scanAIEnrichment(d.QueryRowContext(ctx,
		`SELECT `+aiEnrichmentColumns+` FROM ai_enrichments WHERE book_id = $1 AND user_id = $2 AND status = 'pending' ORDER BY created_at DESC LIMIT 1`,
		bookID, userID,
	))
}

// UpdateAIEnrichmentStatus updates the status of an AI enrichment record.
// Returns ErrInvalidAIEnrichmentStatus for unknown status values.
func (d *DB) UpdateAIEnrichmentStatus(ctx context.Context, userID, id, status string) (*AIEnrichment, error) {
	switch status {
	case AIEnrichmentStatusPending, AIEnrichmentStatusApplied, AIEnrichmentStatusRejected:
	default:
		return nil, fmt.Errorf("invalid ai_enrichment status %q: %w", status, ErrInvalidAIEnrichmentStatus)
	}
	slog.DebugContext(ctx, "db: updating AI enrichment status",
		slog.String(otelkeys.AIEnrichmentID, id),
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Status, status),
	)
	return scanAIEnrichment(d.QueryRowContext(ctx,
		`UPDATE ai_enrichments SET status = $1, updated_at = `+d.now()+` WHERE id = $2 AND user_id = $3 RETURNING `+aiEnrichmentColumns,
		status, id, userID,
	))
}

// DeleteAIEnrichment deletes an AI enrichment record by ID for the given user.
func (d *DB) DeleteAIEnrichment(ctx context.Context, userID, id string) error {
	slog.DebugContext(ctx, "db: deleting AI enrichment",
		slog.String(otelkeys.AIEnrichmentID, id),
		slog.String(otelkeys.UserID, userID),
	)
	res, err := d.ExecContext(ctx,
		`DELETE FROM ai_enrichments WHERE id = $1 AND user_id = $2`,
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

// ApplyAIEnrichmentInput holds the parameters for ApplyAIEnrichment.
type ApplyAIEnrichmentInput struct {
	BookID       string
	UserID       string
	EnrichmentID string
	TagIDs       []string   // union-merged tag IDs to set on the book
	BookUpdate   *BookInput // non-nil when description should be updated
}

// ApplyAIEnrichment atomically sets book tags, optionally updates the book
// description, and marks the enrichment as applied within a single transaction.
// It returns the updated AIEnrichment record.
func (d *DB) ApplyAIEnrichment(ctx context.Context, input ApplyAIEnrichmentInput) (*AIEnrichment, error) {
	slog.DebugContext(ctx, "db: applying AI enrichment",
		slog.String(otelkeys.AIEnrichmentID, input.EnrichmentID),
		slog.String(otelkeys.BookID, input.BookID),
		slog.String(otelkeys.UserID, input.UserID),
	)

	var result *AIEnrichment
	err := d.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Set book tags (delete + re-insert).
		if _, err := tx.ExecContext(ctx, `DELETE FROM book_tags WHERE book_id = $1`, input.BookID); err != nil {
			return fmt.Errorf("delete book tags: %w", err)
		}
		seen := make(map[string]struct{}, len(input.TagIDs))
		for _, tagID := range input.TagIDs {
			if _, ok := seen[tagID]; ok {
				continue
			}
			seen[tagID] = struct{}{}
			if _, err := tx.ExecContext(ctx, `INSERT INTO book_tags (book_id, tag_id) VALUES ($1, $2)`, input.BookID, tagID); err != nil {
				return fmt.Errorf("insert book tag: %w", err)
			}
		}

		// 2. Optionally update book description (only if still empty to avoid overwriting concurrent edits).
		if input.BookUpdate != nil {
			bi := input.BookUpdate
			if _, err := tx.ExecContext(ctx,
				`UPDATE books SET title = $1, description = $2, asin = $3, isbn10 = $4, isbn13 = $5, goodreads_id = $6, hardcover_id = $7, google_books_id = $8, publication_date = $9, publisher = $10, language = $11, cover_image_url = $12, updated_at = `+d.now()+` WHERE id = $13 AND (description IS NULL OR description = '')`,
				bi.Title, bi.Description, bi.ASIN, bi.ISBN10, bi.ISBN13, bi.GoodreadsID, bi.HardcoverID, bi.GoogleBooksID, bi.PublicationDate, bi.Publisher, bi.Language, bi.CoverImageURL, input.BookID,
			); err != nil {
				return fmt.Errorf("update book: %w", err)
			}
		}

		// 3. Mark enrichment as applied (only if still pending).
		row := tx.QueryRowContext(ctx,
			`UPDATE ai_enrichments SET status = $1, updated_at = `+d.now()+` WHERE id = $2 AND user_id = $3 AND status = $4 RETURNING `+aiEnrichmentColumns,
			AIEnrichmentStatusApplied, input.EnrichmentID, input.UserID, AIEnrichmentStatusPending,
		)
		enrichment, err := scanAIEnrichment(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAIEnrichmentNotPending
			}
			return fmt.Errorf("update enrichment status: %w", err)
		}
		result = enrichment
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
