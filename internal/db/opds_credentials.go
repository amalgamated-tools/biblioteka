package db

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ErrOPDSUsernameExists is returned when an OPDS username is already taken by another user.
var ErrOPDSUsernameExists = errors.New("opds username already exists")

// OPDSCredential represents a row in the opds_credentials table.
type OPDSCredential = ProtocolCredential

var opdsCredentialTable = protocolCredentialTable{
	name:              "opds_credentials",
	usernameUniqueCol: "opds_credentials.username",
	usernameUniqueIdx: "idx_opds_credentials_username",
}

// GetOPDSCredentialByUserID returns the OPDS credential for a user, or sql.ErrNoRows if not found.
func (d *DB) GetOPDSCredentialByUserID(ctx context.Context, userID string) (*OPDSCredential, error) {
	slog.DebugContext(ctx, "db: fetching OPDS credential by user ID", slog.String(otelkeys.UserID, userID))
	return d.getProtocolCredentialByUserID(ctx, opdsCredentialTable.name, userID)
}

// GetOPDSCredentialByUsername returns the OPDS credential for a username, or sql.ErrNoRows if not found.
func (d *DB) GetOPDSCredentialByUsername(ctx context.Context, username string) (*OPDSCredential, error) {
	slog.DebugContext(ctx, "db: fetching OPDS credential by username", slog.String(otelkeys.OPDSUsername, username))
	return d.getProtocolCredentialByUsername(ctx, opdsCredentialTable.name, username)
}

// UpsertOPDSCredential creates or updates the OPDS credential for a user.
// Returns ErrOPDSUsernameExists if the username is taken by a different user.
func (d *DB) UpsertOPDSCredential(ctx context.Context, userID, username, passwordHash string) (*OPDSCredential, error) {
	slog.DebugContext(ctx, "db: upserting OPDS credential",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.OPDSUsername, username),
	)
	return d.upsertProtocolCredential(ctx, opdsCredentialTable, userID, username, passwordHash, ErrOPDSUsernameExists)
}

// DeleteOPDSCredential removes the OPDS credential for a user.
func (d *DB) DeleteOPDSCredential(ctx context.Context, userID string) error {
	slog.DebugContext(ctx, "db: deleting OPDS credential", slog.String(otelkeys.UserID, userID))
	return d.deleteProtocolCredential(ctx, opdsCredentialTable.name, userID)
}
