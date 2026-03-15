package db

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Audit action constants for all tracked operations.
const (
	AuditActionLibraryCreated    = "library.created"
	AuditActionLibraryUpdated    = "library.updated"
	AuditActionLibraryDeleted    = "library.deleted"
	AuditActionBookCreated       = "book.created"
	AuditActionBookUpdated       = "book.updated"
	AuditActionBookDeleted       = "book.deleted"
	AuditActionAuthorCreated     = "author.created"
	AuditActionAuthorUpdated     = "author.updated"
	AuditActionAuthorDeleted     = "author.deleted"
	AuditActionSeriesCreated     = "series.created"
	AuditActionSeriesUpdated     = "series.updated"
	AuditActionSeriesDeleted     = "series.deleted"
	AuditActionBookFileCreated   = "book_file.created"
	AuditActionBookFileDeleted   = "book_file.deleted"
	AuditActionUserSignedUp      = "user.signed_up"
	AuditActionAdminUpdated      = "user.admin_updated"
	AuditActionSMTPConfigUpdated = "smtp.config_updated"
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

func scanAuditLog(row interface{ Scan(...any) error }) (*AuditLog, error) {
	var entry AuditLog
	err := row.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.Metadata, &entry.CreatedAt)
	if err != nil {
		return nil, err
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
			return err
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
	return err
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
		return nil, 0, err
	}

	rows, err := d.QueryContext(ctx,
		`SELECT `+auditLogColumns+` FROM audit_logs ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditLog
	for rows.Next() {
		entry, err := scanAuditLog(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
