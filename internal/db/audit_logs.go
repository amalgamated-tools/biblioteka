package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Audit action constants for all tracked operations.
const (
	AuditActionAdminUpdated            = "user.admin_updated"
	AuditActionAuthorCreated           = "author.created"
	AuditActionAuthorDeleted           = "author.deleted"
	AuditActionAuthorUpdated           = "author.updated"
	AuditActionBookCreated             = "book.created"
	AuditActionBookDeleted             = "book.deleted"
	AuditActionBookFileCreated         = "book_file.created"
	AuditActionBookFileDeleted         = "book_file.deleted"
	AuditActionBookUpdated             = "book.updated"
	AuditActionAPIKeyCreated           = "api_key.created"
	AuditActionAPIKeyDeleted           = "api_key.deleted"
	AuditActionKoboTokenCreated        = "kobo_token.created"
	AuditActionKoboTokenDeleted        = "kobo_token.deleted"
	AuditActionLibraryCreated          = "library.created"
	AuditActionLibraryDeleted          = "library.deleted"
	AuditActionLibraryUpdated          = "library.updated"
	AuditActionOPDSCredentialDeleted   = "opds_credential.deleted"
	AuditActionOPDSCredentialUpdated   = "opds_credential.updated"
	AuditActionKOSyncCredentialDeleted = "kosync_credential.deleted"
	AuditActionKOSyncCredentialUpdated = "kosync_credential.updated"
	AuditActionSeriesCreated           = "series.created"
	AuditActionSeriesDeleted           = "series.deleted"
	AuditActionSeriesUpdated           = "series.updated"
	AuditActionSMTPConfigUpdated       = "smtp.config_updated"
	AuditActionUserSignedUp            = "user.signed_up"
)

// AuditLog represents a single audit log entry.
type AuditLog struct {
	ID         string    `json:"id"`
	UserID     *string   `json:"user_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Metadata   *string   `json:"metadata"`
	CreatedAt  Timestamp `json:"created_at"`
}

const auditLogColumns = `id, user_id, action, entity_type, entity_id, metadata, created_at`

func scanAuditLog(ctx context.Context, row interface{ Scan(...any) error }) (*AuditLog, error) {
	var entry AuditLog
	err := row.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.Metadata, &entry.CreatedAt)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to scan audit log", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to scan audit log: %w", err)
	}
	return &entry, nil
}

// CreateAuditLog inserts a new audit log entry. The metadata map is serialised
// to JSON; a nil map stores a NULL metadata value. The userID may be empty for
// system-initiated actions.
func (d *DB) CreateAuditLog(ctx context.Context, userID, action, entityType, entityID string, metadata map[string]any) error {
	slog.DebugContext(ctx, "db: creating audit log",
		slog.String(otelkeys.Action, action),
		slog.String(otelkeys.EntityType, entityType),
		slog.String(otelkeys.EntityID, entityID),
	)

	var metadataJSON *string
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal audit log metadata", slog.Any(otelkeys.Error, err))
			return fmt.Errorf("failed to marshal audit log metadata: %w", err)
		}
		s := string(b)
		metadataJSON = &s
	}

	var uid *string
	if userID != "" {
		uid = &userID
	}

	_, err := d.ExecContext(ctx,
		`INSERT INTO audit_logs (user_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, $3, $4, $5)`,
		uid, action, entityType, entityID, metadataJSON,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create audit log", slog.Any(otelkeys.Error, err))
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// ListAuditLogs returns audit log entries ordered by creation time (newest first),
// with the total count of all entries. limit and offset control pagination.
func (d *DB) ListAuditLogs(ctx context.Context, limit, offset int) ([]AuditLog, int, error) {
	slog.DebugContext(ctx, "db: listing audit logs",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "Failed to count audit logs", slog.Any(otelkeys.Error, err))
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	rows, err := d.QueryContext(ctx,
		`SELECT `+auditLogColumns+` FROM audit_logs ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to query audit logs", slog.Any(otelkeys.Error, err))
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var entries []AuditLog
	for rows.Next() {
		entry, err := scanAuditLog(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan audit log from list", slog.Any(otelkeys.Error, err))
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		entries = append(entries, *entry)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate audit log rows", slog.Any(otelkeys.Error, err))
		return nil, 0, fmt.Errorf("failed to iterate audit log rows: %w", err)
	}
	return entries, total, nil
}
