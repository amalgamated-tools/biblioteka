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

// lookupProtocolCred performs a by-username DB lookup with consistent
// debug/error logging and error wrapping. notFoundMsg is logged at debug
// level when sql.ErrNoRows is returned; failedMsg is used for both the error
// log and the wrapped error returned to the caller.
func lookupProtocolCred(
	ctx context.Context,
	username string,
	usernameAttr slog.Attr,
	notFoundMsg, failedMsg string,
	lookup func(context.Context, string) (*db.ProtocolCredential, error),
) (*auth.ProtocolCredentialResult, error) {
	cred, err := lookup(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, notFoundMsg, usernameAttr)
		} else {
			slog.ErrorContext(ctx, failedMsg, usernameAttr, slog.Any(otelkeys.Error, err))
		}
		return nil, fmt.Errorf("%s for username %s: %w", failedMsg, username, err)
	}
	return &auth.ProtocolCredentialResult{
		UserID:       cred.UserID,
		PasswordHash: cred.PasswordHash,
	}, nil
}

// GetOPDSCredential looks up the OPDS credential for the given username and
// returns the associated user ID and bcrypt-hashed password for the auth
// middleware to verify. Returns sql.ErrNoRows (wrapped) when not found.
func (a *protocolCredDBAdapter) GetOPDSCredential(ctx context.Context, username string) (*auth.ProtocolCredentialResult, error) {
	return lookupProtocolCred(ctx, username,
		slog.String(otelkeys.OPDSUsername, username),
		"OPDS credential not found", "failed to get OPDS credential",
		a.db.GetOPDSCredentialByUsername,
	)
}

// GetKOSyncCredential looks up the KOSync credential for the given username
// and returns the associated user ID and bcrypt-hashed password for the auth
// middleware to verify. Returns sql.ErrNoRows (wrapped) when not found.
func (a *protocolCredDBAdapter) GetKOSyncCredential(ctx context.Context, username string) (*auth.ProtocolCredentialResult, error) {
	return lookupProtocolCred(ctx, username,
		slog.String(otelkeys.KOSyncUsername, username),
		"KOSync credential not found", "failed to get KOSync credential",
		a.db.GetKOSyncCredentialByUsername,
	)
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
