package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/goauth/auth"
	goauthhandler "github.com/amalgamated-tools/goauth/handler"
	"github.com/stretchr/testify/require"

	"github.com/amalgamated-tools/biblioteka/internal/authstore"
	internaldb "github.com/amalgamated-tools/biblioteka/internal/db"
)

// stubAPIKeyStore is a minimal implementation of auth.APIKeyStore for testing
// the method dispatch wrappers. Only the methods called during tests need real
// implementations; others return zero values.
type stubAPIKeyStore struct{}

func (s *stubAPIKeyStore) CreateAPIKey(context.Context, string, string, string, string) (*auth.APIKey, error) {
	return nil, nil
}

func (s *stubAPIKeyStore) ListAPIKeysByUser(context.Context, string) ([]auth.APIKey, error) {
	return nil, nil
}

func (s *stubAPIKeyStore) FindAPIKeyByIDAndUser(context.Context, string, string) (*auth.APIKey, error) {
	return nil, nil
}

func (s *stubAPIKeyStore) ValidateAPIKey(context.Context, string) (string, string, error) {
	return "", "", nil
}

func (s *stubAPIKeyStore) TouchAPIKeyLastUsed(context.Context, string) error {
	return nil
}

func (s *stubAPIKeyStore) DeleteAPIKey(_ context.Context, id, _ string) error {
	if id == "" {
		return sql.ErrNoRows
	}
	return nil
}

func newTestAPIKeyHandler() *APIKeyHandler {
	return &APIKeyHandler{
		APIKeyHandler: goauthhandler.APIKeyHandler{
			APIKeys: &stubAPIKeyStore{},
			Prefix:  "bib_",
			URLParamFunc: func(r *http.Request, _ string) string {
				rest := strings.TrimPrefix(r.URL.Path, "/api/api-keys/")
				rest = strings.TrimSuffix(rest, "/")
				if strings.Contains(rest, "/") {
					return ""
				}
				return rest
			},
		},
	}
}

func TestHandleAPIKeys_MethodNotAllowed(t *testing.T) {
	h := newTestAPIKeyHandler()

	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/api-keys", nil)
			rec := httptest.NewRecorder()

			h.HandleAPIKeys(rec, req)

			require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestHandleAPIKeys_GET(t *testing.T) {
	h := newTestAPIKeyHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	rec := httptest.NewRecorder()

	h.HandleAPIKeys(rec, req)

	// List returns empty JSON array with 200 (stubbed store returns nil slice).
	require.Equal(t, http.StatusOK, rec.Code)
	var body []json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Empty(t, body)
}

func TestHandleAPIKey_MethodNotAllowed(t *testing.T) {
	h := newTestAPIKeyHandler()

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/api-keys/some-id", nil)
			rec := httptest.NewRecorder()

			h.HandleAPIKey(rec, req)

			require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestHandleAPIKey_DELETE_ExtractsID(t *testing.T) {
	h := newTestAPIKeyHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/api-keys/test-key-id", nil)
	rec := httptest.NewRecorder()

	h.HandleAPIKey(rec, req)

	// The stub store returns nil for valid IDs — expect 204 No Content.
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandleAPIKey_DELETE_EmptyID(t *testing.T) {
	h := newTestAPIKeyHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/api-keys/", nil)
	rec := httptest.NewRecorder()

	h.HandleAPIKey(rec, req)

	// URLParamFunc returns "" for trailing-slash path; goauth returns 400.
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// stubAPIKeyStoreEmptyID returns an APIKey with an empty ID to simulate a
// malformed goauth response.
type stubAPIKeyStoreEmptyID struct{ stubAPIKeyStore }

func (s *stubAPIKeyStoreEmptyID) CreateAPIKey(_ context.Context, _, name, _, _ string) (*auth.APIKey, error) {
	return &auth.APIKey{ID: "", Name: name}, nil
}

func TestAPIKeyHandler_Create_EmptyIDProducesNoAudit(t *testing.T) {
	d := newTestDB(t)

	h := &APIKeyHandler{
		APIKeyHandler: goauthhandler.APIKeyHandler{
			APIKeys: &stubAPIKeyStoreEmptyID{},
			Prefix:  "bib_",
			URLParamFunc: func(r *http.Request, _ string) string {
				return strings.TrimPrefix(r.URL.Path, "/api/api-keys/")
			},
		},
		DB: d,
	}

	body := `{"name":"My Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/api-keys", strings.NewReader(body))
	req = withUserID(req, "user-123")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	logs, _, err := d.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err)
	require.Empty(t, logs, "expected no audit entry when goauth returns empty ID")
}

func TestAuthHandler_Signup_BlockedByDBSetting(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, d.SetSetting(t.Context(), internaldb.SettingRegistrationDisabled, "true"))

	h := &AuthHandler{DB: d}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(`{"name":"Alice","email":"alice@example.com","password":"password1"}`))
	rec := httptest.NewRecorder()

	h.Signup(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuthHandler_Signup_NotBlockedWhenSettingFalse(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, d.SetSetting(t.Context(), internaldb.SettingRegistrationDisabled, "false"))

	// Wire up a real Users store so the signup can proceed past the DB check.
	h := &AuthHandler{
		AuthHandler: goauthhandler.AuthHandler{
			Users: &authstore.UserAdapter{DB: d},
			JWT:   newTestJWT(t),
		},
		DB: d,
	}

	body := `{"name":"Alice","email":"alice@example.com","password":"password1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Signup(rec, req)

	// Should pass the DB check; goauth handles the actual signup.
	require.NotEqual(t, http.StatusForbidden, rec.Code, "signup should not be blocked when registration_disabled=false")
}

func TestOIDCHandler_Callback_BlockedByRegistrationDisabled(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, d.SetSetting(t.Context(), internaldb.SettingRegistrationDisabled, "true"))

	h := &OIDCHandler{DB: d}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback", nil)
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}
