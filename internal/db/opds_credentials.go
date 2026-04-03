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

var opdsCredConfig = protocolCredentialConfig{
	table:        "opds_credentials",
	tableCol:     "opds_credentials.username",
	indexName:    "idx_opds_credentials_username",
	errExists:    ErrOPDSUsernameExists,
	logPrefix:    "OPDS",
	usernameAttr: func(v string) slog.Attr { return slog.String(otelkeys.OPDSUsername, v) },
}

// GetOPDSCredentialByUserID returns the OPDS credential for a user, or sql.ErrNoRows if not found.
func (d *DB) GetOPDSCredentialByUserID(ctx context.Context, userID string) (*OPDSCredential, error) {
	return getCredentialByUserID(ctx, d, opdsCredConfig, userID)
}

// GetOPDSCredentialByUsername returns the OPDS credential for a username, or sql.ErrNoRows if not found.
func (d *DB) GetOPDSCredentialByUsername(ctx context.Context, username string) (*OPDSCredential, error) {
	return getCredentialByUsername(ctx, d, opdsCredConfig, username)
}

// UpsertOPDSCredential creates or updates the OPDS credential for a user.
// Returns ErrOPDSUsernameExists if the username is taken by a different user.
func (d *DB) UpsertOPDSCredential(ctx context.Context, userID, username, passwordHash string) (*OPDSCredential, error) {
	return upsertCredential(ctx, d, opdsCredConfig, userID, username, passwordHash)
}

// DeleteOPDSCredential removes the OPDS credential for a user.
func (d *DB) DeleteOPDSCredential(ctx context.Context, userID string) error {
	return deleteCredential(ctx, d, opdsCredConfig, userID)
}
