package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// protocolCredDBAdapter bridges *db.DB to the auth.OPDSCredentialChecker
// and auth.KOSyncCredentialChecker interfaces.
type protocolCredDBAdapter struct {
	db *db.DB
}

// GetOPDSCredential looks up the OPDS credential for the given username and
// returns the associated user ID and bcrypt-hashed password for the auth
// middleware to verify. Returns sql.ErrNoRows (wrapped) when not found.
func (a *protocolCredDBAdapter) GetOPDSCredential(ctx context.Context, username string) (*auth.ProtocolCredentialResult, error) {
	cred, err := a.db.GetOPDSCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(
				ctx,
				"OPDS credential not found",
				slog.String(otelkeys.OPDSUsername, username),
			)
		} else {
			slog.ErrorContext(
				ctx,
				"failed to get OPDS credential",
				slog.String(otelkeys.OPDSUsername, username),
				slog.Any(otelkeys.Error, err),
			)
		}
		return nil, fmt.Errorf("failed to get OPDS credential for username %s: %w", username, err)
	}
	return &auth.ProtocolCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}

// GetKOSyncCredential looks up the KOSync credential for the given username
// and returns the associated user ID and bcrypt-hashed password for the auth
// middleware to verify. Returns sql.ErrNoRows (wrapped) when not found.
func (a *protocolCredDBAdapter) GetKOSyncCredential(ctx context.Context, username string) (*auth.ProtocolCredentialResult, error) {
	cred, err := a.db.GetKOSyncCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(
				ctx,
				"KOSync credential not found",
				slog.String(otelkeys.KOSyncUsername, username),
			)
		} else {
			slog.ErrorContext(
				ctx,
				"failed to get KOSync credential",
				slog.String(otelkeys.KOSyncUsername, username),
				slog.Any(otelkeys.Error, err),
			)
		}
		return nil, fmt.Errorf("failed to get KOSync credential for username %s: %w", username, err)
	}
	return &auth.ProtocolCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}

// koboDBAdapter bridges *db.DB to the auth.KoboTokenChecker interface.
type koboDBAdapter struct {
	db *db.DB
}

// GetKoboTokenByToken hashes the raw Kobo token and looks up the matching
// record, returning the associated user ID for injection into the request
// context by KoboAuthMiddleware. Returns sql.ErrNoRows (wrapped) when the
// token is not found.
func (a *koboDBAdapter) GetKoboTokenByToken(ctx context.Context, token string) (*auth.KoboTokenResult, error) {
	tokenHash := auth.HashKoboToken(token)
	t, err := a.db.GetKoboTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(
				ctx,
				"Kobo token not found",
				slog.String(otelkeys.TokenHash, tokenHash),
			)
		} else {
			slog.ErrorContext(
				ctx,
				"failed to get Kobo token",
				slog.String(otelkeys.TokenHash, tokenHash),
				slog.Any(otelkeys.Error, err),
			)
		}
		return nil, fmt.Errorf("failed to get Kobo token for token hash %s: %w", tokenHash, err)
	}
	return &auth.KoboTokenResult{
		UserID: t.UserID,
	}, nil
}
