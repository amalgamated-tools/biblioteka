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
		e.SuggestedTags = []string{}
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
		slog.String(otelkeys.AIEnrichmentID, ""),
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
