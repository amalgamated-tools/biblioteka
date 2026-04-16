// Package authstore provides adapters that bridge biblioteka's db.DB
// to the goauth store interfaces.
package authstore

import (
	"context"
	"errors"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/goauth/auth"
)

// UserAdapter wraps *db.DB and implements auth.UserStore.
type UserAdapter struct {
	DB *db.DB
}

func dbUserToAuth(u *db.User) *auth.User {
	if u == nil {
		return nil
	}
	return &auth.User{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		OIDCSubject:  u.OIDCSubject,
		IsAdmin:      u.IsAdmin,
		CreatedAt:    u.CreatedAt.Time,
	}
}

func (a *UserAdapter) CreateUser(ctx context.Context, name, email, passwordHash string) (*auth.User, error) {
	u, err := a.DB.CreateUser(ctx, name, email, passwordHash)
	if errors.Is(err, db.ErrEmailExists) {
		return nil, auth.ErrEmailExists
	}
	return dbUserToAuth(u), err
}

func (a *UserAdapter) CreateOIDCUser(ctx context.Context, name, email, oidcSubject string) (*auth.User, error) {
	u, err := a.DB.CreateOIDCUser(ctx, name, email, oidcSubject)
	if errors.Is(err, db.ErrEmailExists) {
		return nil, auth.ErrEmailExists
	}
	return dbUserToAuth(u), err
}

func (a *UserAdapter) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	u, err := a.DB.GetUserByEmail(ctx, email)
	return dbUserToAuth(u), err
}

func (a *UserAdapter) FindByID(ctx context.Context, id string) (*auth.User, error) {
	u, err := a.DB.GetUserByID(ctx, id)
	return dbUserToAuth(u), err
}

func (a *UserAdapter) FindByOIDCSubject(ctx context.Context, subject string) (*auth.User, error) {
	u, err := a.DB.GetUserByOIDCSubject(ctx, subject)
	return dbUserToAuth(u), err
}

func (a *UserAdapter) LinkOIDCSubject(ctx context.Context, userID, oidcSubject string) error {
	return a.DB.LinkOIDCSubject(ctx, userID, oidcSubject)
}

func (a *UserAdapter) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return a.DB.UpdatePassword(ctx, userID, passwordHash)
}

func (a *UserAdapter) UpdateName(ctx context.Context, userID, name string) (*auth.User, error) {
	u, err := a.DB.UpdateName(ctx, userID, name)
	return dbUserToAuth(u), err
}

func (a *UserAdapter) IsAdmin(ctx context.Context, userID string) (bool, error) {
	return a.DB.IsAdmin(ctx, userID)
}

func (a *UserAdapter) CountUsers(ctx context.Context) (int, error) {
	return a.DB.CountUsers(ctx)
}

// APIKeyAdapter wraps *db.DB and implements auth.APIKeyStore.
type APIKeyAdapter struct {
	DB *db.DB
}

func dbAPIKeyToAuth(k *db.APIKey) *auth.APIKey {
	if k == nil {
		return nil
	}
	ak := &auth.APIKey{
		ID:        k.ID,
		UserID:    k.UserID,
		Name:      k.Name,
		KeyHash:   k.KeyHash,
		KeyPrefix: k.KeyPrefix,
		CreatedAt: k.CreatedAt.Time,
	}
	if k.LastUsedAt != nil {
		t := k.LastUsedAt.Time
		ak.LastUsedAt = &t
	}
	return ak
}

func (a *APIKeyAdapter) CreateAPIKey(ctx context.Context, userID, name, keyHash, keyPrefix string) (*auth.APIKey, error) {
	k, err := a.DB.CreateAPIKey(ctx, userID, name, keyHash, keyPrefix)
	return dbAPIKeyToAuth(k), err
}

func (a *APIKeyAdapter) ListAPIKeysByUser(ctx context.Context, userID string) ([]auth.APIKey, error) {
	keys, err := a.DB.ListAPIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]auth.APIKey, len(keys))
	for i := range keys {
		result[i] = *dbAPIKeyToAuth(&keys[i])
	}
	return result, nil
}

func (a *APIKeyAdapter) FindAPIKeyByIDAndUser(ctx context.Context, id, userID string) (*auth.APIKey, error) {
	k, err := a.DB.GetAPIKey(ctx, id, userID)
	return dbAPIKeyToAuth(k), err
}

func (a *APIKeyAdapter) ValidateAPIKey(ctx context.Context, keyHash string) (string, string, error) {
	return a.DB.ValidateAPIKey(ctx, keyHash)
}

func (a *APIKeyAdapter) TouchAPIKeyLastUsed(ctx context.Context, id string) error {
	return a.DB.TouchAPIKeyLastUsed(ctx, id)
}

func (a *APIKeyAdapter) DeleteAPIKey(ctx context.Context, id, userID string) error {
	return a.DB.DeleteAPIKey(ctx, id, userID)
}

// PasskeyAdapter wraps *db.DB and implements auth.PasskeyStore.
type PasskeyAdapter struct {
	DB *db.DB
}

func (a *PasskeyAdapter) CreateChallenge(ctx context.Context, userID *string, sessionData string, expiresAt time.Time) (*auth.PasskeyChallenge, error) {
	c, err := a.DB.CreatePasskeyChallenge(ctx, userID, sessionData, expiresAt)
	if err != nil {
		return nil, err
	}
	return &auth.PasskeyChallenge{
		ID: c.ID, UserID: c.UserID, SessionData: c.SessionData,
		ExpiresAt: c.ExpiresAt.Time, CreatedAt: c.CreatedAt.Time,
	}, nil
}

func (a *PasskeyAdapter) GetAndDeleteChallenge(ctx context.Context, id string) (*auth.PasskeyChallenge, error) {
	c, err := a.DB.GetAndDeletePasskeyChallenge(ctx, id)
	if err != nil {
		return nil, err
	}
	return &auth.PasskeyChallenge{
		ID: c.ID, UserID: c.UserID, SessionData: c.SessionData,
		ExpiresAt: c.ExpiresAt.Time, CreatedAt: c.CreatedAt.Time,
	}, nil
}

func (a *PasskeyAdapter) DeleteExpiredChallenges(ctx context.Context) error {
	return a.DB.DeleteExpiredPasskeyChallenges(ctx)
}

func (a *PasskeyAdapter) CreateCredential(ctx context.Context, userID, name, credentialID, credentialData, aaguid string) (*auth.PasskeyCredential, error) {
	c, err := a.DB.CreatePasskeyCredential(ctx, userID, name, credentialID, credentialData, aaguid)
	if err != nil {
		return nil, err
	}
	return &auth.PasskeyCredential{
		ID: c.ID, UserID: c.UserID, Name: c.Name,
		CredentialID: c.CredentialID, CredentialData: c.CredentialData,
		AAGUID: c.AAGUID, CreatedAt: c.CreatedAt.Time,
	}, nil
}

func (a *PasskeyAdapter) ListCredentialsByUser(ctx context.Context, userID string) ([]auth.PasskeyCredential, error) {
	creds, err := a.DB.ListPasskeyCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]auth.PasskeyCredential, len(creds))
	for i, c := range creds {
		result[i] = auth.PasskeyCredential{
			ID: c.ID, UserID: c.UserID, Name: c.Name,
			CredentialID: c.CredentialID, CredentialData: c.CredentialData,
			AAGUID: c.AAGUID, CreatedAt: c.CreatedAt.Time,
		}
	}
	return result, nil
}

func (a *PasskeyAdapter) FindCredentialByCredentialID(ctx context.Context, credentialID string) (*auth.PasskeyCredential, error) {
	c, err := a.DB.GetPasskeyCredentialByCredentialID(ctx, credentialID)
	if err != nil {
		return nil, err
	}
	return &auth.PasskeyCredential{
		ID: c.ID, UserID: c.UserID, Name: c.Name,
		CredentialID: c.CredentialID, CredentialData: c.CredentialData,
		AAGUID: c.AAGUID, CreatedAt: c.CreatedAt.Time,
	}, nil
}

func (a *PasskeyAdapter) FindCredentialByIDAndUser(ctx context.Context, id, userID string) (*auth.PasskeyCredential, error) {
	c, err := a.DB.GetPasskeyCredential(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return &auth.PasskeyCredential{
		ID: c.ID, UserID: c.UserID, Name: c.Name,
		CredentialID: c.CredentialID, CredentialData: c.CredentialData,
		AAGUID: c.AAGUID, CreatedAt: c.CreatedAt.Time,
	}, nil
}

func (a *PasskeyAdapter) UpdateCredentialData(ctx context.Context, userID, credentialID, credentialData string) error {
	return a.DB.UpdatePasskeyCredentialData(ctx, userID, credentialID, credentialData)
}

func (a *PasskeyAdapter) DeleteCredential(ctx context.Context, id, userID string) error {
	return a.DB.DeletePasskeyCredential(ctx, id, userID)
}
