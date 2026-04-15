package db

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Audit action constants for all tracked operations.
const (
	AuditActionAdminUpdated            = "user.admin_updated"
	AuditActionAnnotationCreated       = "annotation.created"
	AuditActionAnnotationDeleted       = "annotation.deleted"
	AuditActionAnnotationUpdated       = "annotation.updated"
	AuditActionAPIKeyCreated           = "api_key.created"
	AuditActionAPIKeyDeleted           = "api_key.deleted"
	AuditActionAuthorCreated           = "author.created"
	AuditActionAuthorDeleted           = "author.deleted"
	AuditActionAuthorUpdated           = "author.updated"
	AuditActionBookCreated             = "book.created"
	AuditActionBookDeleted             = "book.deleted"
	AuditActionCalibreImported         = "calibre.imported"
	AuditActionBookFileCreated         = "book_file.created"
	AuditActionBookFileDeleted         = "book_file.deleted"
	AuditActionBookFileEmailed         = "book_file.emailed"
	AuditActionBookUpdated             = "book.updated"
	AuditActionBookUploaded            = "book.uploaded"
	AuditActionFTSRebuilt              = "fts.rebuilt"
	AuditActionGroupCreated            = "group.created"
	AuditActionGroupDeleted            = "group.deleted"
	AuditActionGroupListShared         = "group.list_shared"
	AuditActionGroupListUnshared       = "group.list_unshared"
	AuditActionGroupMemberAdded        = "group.member_added"
	AuditActionGroupMemberRemoved      = "group.member_removed"
	AuditActionGroupUpdated            = "group.updated"
	AuditActionKoboTokenCreated        = "kobo_token.created"
	AuditActionKoboTokenDeleted        = "kobo_token.deleted"
	AuditActionKOSyncCredentialDeleted = "kosync_credential.deleted"
	AuditActionKOSyncCredentialUpdated = "kosync_credential.updated"
	AuditActionLibraryCreated          = "library.created"
	AuditActionLibraryDeleted          = "library.deleted"
	AuditActionLibraryUpdated          = "library.updated"
	AuditActionMetadataApplied         = "metadata.applied"
	AuditActionMetadataFetchRequested  = "metadata.fetch_requested"
	AuditActionMetadataRejected        = "metadata.rejected"
	AuditActionOPDSCredentialDeleted   = "opds_credential.deleted"
	AuditActionOPDSCredentialUpdated   = "opds_credential.updated"
	AuditActionPasskeyCreated          = "passkey.created"
	AuditActionPasskeyDeleted          = "passkey.deleted"
	AuditActionPasswordChanged         = "user.password_changed"
	AuditActionReadingListBookAdded    = "reading_list.book_added"
	AuditActionReadingListBookRemoved  = "reading_list.book_removed"
	AuditActionReadingListCreated      = "reading_list.created"
	AuditActionReadingListDeleted      = "reading_list.deleted"
	AuditActionReadingListUpdated      = "reading_list.updated"
	AuditActionSeriesCreated           = "series.created"
	AuditActionSeriesDeleted           = "series.deleted"
	AuditActionSeriesUpdated           = "series.updated"
	AuditActionSMTPConfigUpdated       = "smtp.config_updated"
	AuditActionUserProfileUpdated      = "user.profile_updated"
	AuditActionUserSignedUp            = "user.signed_up"
	AuditActionWatchFolderUpdated      = "watch_folder.config_updated"
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

// scanAuditLogAndTotal scans audit log columns plus a trailing COUNT(*) OVER() total.
func scanAuditLogAndTotal(row interface{ Scan(...any) error }) (*AuditLog, int, error) {
	var entry AuditLog
	var total int
	err := row.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.Metadata, &entry.CreatedAt, &total)
	if err != nil {
		return nil, 0, err
	}
	return &entry, total, nil
}

// CreateAuditLog inserts a new audit log entry. The metadata map is serialised
// to JSON; a nil map stores a NULL metadata value. The userID may be empty for
// system-initiated actions.
func (d *DB) CreateAuditLog(ctx context.Context, userID, action, entityType, entityID string, metadata map[string]any) error {
	slog.DebugContext(ctx, "creating audit log",
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
// A single query with COUNT(*) OVER() is used to avoid a separate COUNT round-trip.
// When limit <= 0 the query would return zero rows and the window function would
// produce no total; in that case a standalone COUNT(*) is issued instead and an
// empty slice is returned with the correct total.
func (d *DB) ListAuditLogs(ctx context.Context, limit, offset int) ([]AuditLog, int, error) {
	slog.DebugContext(ctx, "listing audit logs",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	if limit <= 0 {
		var total int
		if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
			return nil, 0, err
		}
		return []AuditLog{}, total, nil
	}

	rows, err := d.QueryContext(ctx,
		`SELECT `+auditLogColumns+`, COUNT(*) OVER() AS total FROM audit_logs ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	entries, total, err := collectRowsAndTotal(rows, scanAuditLogAndTotal)
	if err != nil {
		return nil, 0, err
	}

	// If no rows were returned at a non-zero offset, the total is 0 from
	// collectRowsAndTotal but the real count may be non-zero. Fall back to a
	// COUNT(*) query so callers can still compute the correct page count.
	if err := countFallback(ctx, d, &total, len(entries), offset, `SELECT COUNT(*) FROM audit_logs`); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}
