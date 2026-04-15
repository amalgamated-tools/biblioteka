package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

// newTestPasskeyHandler creates a PasskeyHandler wired to an in-memory SQLite DB
// and a real WebAuthn instance configured for localhost testing.
func newTestPasskeyHandler(t *testing.T) (*PasskeyHandler, *db.DB) {
	t.Helper()
	database := newTestDB(t)
	jwt := newTestJWT(t)

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost"},
	})
	require.NoError(t, err)

	return &PasskeyHandler{
		DB:            database,
		WebAuthn:      wa,
		JWT:           jwt,
		SecureCookies: false,
	}, database
}

func TestPasskeyHandler_HandlePasskeyEnabled(t *testing.T) {
	t.Run("enabled when WebAuthn configured", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodGet, "/api/auth/passkey/enabled", nil)
		w := httptest.NewRecorder()
		h.HandlePasskeyEnabled(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]bool
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.True(t, resp["enabled"])
	})

	t.Run("disabled when WebAuthn nil", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)
		h.WebAuthn = nil

		r := httptest.NewRequest(http.MethodGet, "/api/auth/passkey/enabled", nil)
		w := httptest.NewRecorder()
		h.HandlePasskeyEnabled(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]bool
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.False(t, resp["enabled"])
	})

	t.Run("method not allowed for non-GET", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/enabled", nil)
		w := httptest.NewRecorder()
		h.HandlePasskeyEnabled(w, r)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestPasskeyHandler_HandlePasskeyCredentials(t *testing.T) {
	t.Run("list returns empty slice for new user", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Alice", "alice@example.com", "hash")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodGet, "/api/auth/passkey/credentials", nil)
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandlePasskeyCredentials(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		var creds []passkeyCredentialDTO
		require.NoError(t, json.NewDecoder(w.Body).Decode(&creds))
		require.Empty(t, creds)
	})

	t.Run("list returns stored credentials", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Bob", "bob@example.com", "hash")
		require.NoError(t, err)

		_, err = database.CreatePasskeyCredential(t.Context(), user.ID, "My Key", "cred-1", `{}`, "aaguid-1")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodGet, "/api/auth/passkey/credentials", nil)
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandlePasskeyCredentials(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		var creds []passkeyCredentialDTO
		require.NoError(t, json.NewDecoder(w.Body).Decode(&creds))
		require.Len(t, creds, 1)
		require.Equal(t, "My Key", creds[0].Name)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Cath", "cath@example.com", "hash")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodDelete, "/api/auth/passkey/credentials", nil)
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandlePasskeyCredentials(w, r)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestPasskeyHandler_HandlePasskeyCredential_Delete(t *testing.T) {
	t.Run("delete own credential", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Dave", "dave@example.com", "hash")
		require.NoError(t, err)

		cred, err := database.CreatePasskeyCredential(t.Context(), user.ID, "Dave Key", "cred-dave", `{}`, "")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodDelete, "/api/auth/passkey/credentials/"+cred.ID, nil)
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandlePasskeyCredential(w, r)

		require.Equal(t, http.StatusNoContent, w.Code)

		_, err = database.GetPasskeyCredential(t.Context(), cred.ID, user.ID)
		require.Error(t, err)
	})

	t.Run("delete other user's credential returns 404", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		owner, err := database.CreateUser(t.Context(), "Eve", "eve@example.com", "hash")
		require.NoError(t, err)
		other, err := database.CreateUser(t.Context(), "Frank", "frank@example.com", "hash2")
		require.NoError(t, err)

		cred, err := database.CreatePasskeyCredential(t.Context(), owner.ID, "Eve Key", "cred-eve", `{}`, "")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodDelete, "/api/auth/passkey/credentials/"+cred.ID, nil)
		r = withUserID(r, other.ID)
		w := httptest.NewRecorder()
		h.HandlePasskeyCredential(w, r)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodDelete, "/api/auth/passkey/credentials/", nil)
		w := httptest.NewRecorder()
		h.HandlePasskeyCredential(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPasskeyHandler_BeginRegistration(t *testing.T) {
	t.Run("returns session_id and options", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Grace", "grace@example.com", "hash")
		require.NoError(t, err)

		body := `{"name":"My iPhone"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/begin", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleBeginRegistration(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		var resp passkeyBeginResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.NotEmpty(t, resp.SessionID)
		require.NotNil(t, resp.Options)
	})

	t.Run("blank name returns 400", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Hank", "hank@example.com", "hash")
		require.NoError(t, err)

		body := `{"name":"   "}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/begin", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleBeginRegistration(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("name too long returns 400", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Hans", "hans@example.com", "hash")
		require.NoError(t, err)

		longName := strings.Repeat("a", maxTokenNameLength+1)
		body := `{"name":"` + longName + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/begin", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleBeginRegistration(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("passkeys disabled returns 503", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		h.WebAuthn = nil
		user, err := database.CreateUser(t.Context(), "Iris", "iris@example.com", "hash")
		require.NoError(t, err)

		body := `{"name":"Key"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/begin", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleBeginRegistration(w, r)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Jane", "jane@example.com", "hash")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodGet, "/api/auth/passkey/register/begin", nil)
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleBeginRegistration(w, r)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestPasskeyHandler_BeginAuthentication(t *testing.T) {
	t.Run("returns session_id and options", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/begin", nil)
		w := httptest.NewRecorder()
		h.HandleBeginAuthentication(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		var resp passkeyBeginResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.NotEmpty(t, resp.SessionID)
		require.NotNil(t, resp.Options)
	})

	t.Run("passkeys disabled returns 503", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)
		h.WebAuthn = nil

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/begin", nil)
		w := httptest.NewRecorder()
		h.HandleBeginAuthentication(w, r)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodGet, "/api/auth/passkey/login/begin", nil)
		w := httptest.NewRecorder()
		h.HandleBeginAuthentication(w, r)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestPasskeyHandler_FinishRegistration_InvalidSession(t *testing.T) {
	t.Run("missing session_id returns 400", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Kim", "kim@example.com", "hash")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/finish", strings.NewReader(`{}`))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleFinishRegistration(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unknown session_id returns 400", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Leo", "leo@example.com", "hash")
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/finish?session_id=does-not-exist", strings.NewReader(`{}`))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleFinishRegistration(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("expired session returns 400", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		user, err := database.CreateUser(t.Context(), "Mia", "mia@example.com", "hash")
		require.NoError(t, err)

		uid := user.ID
		past := time.Now().UTC().Add(-time.Minute)
		data := passkeyChallengeData{SessionData: webauthn.SessionData{}, Name: "key"}
		enc, err := json.Marshal(data)
		require.NoError(t, err)
		ch, err := database.CreatePasskeyChallenge(t.Context(), &uid, string(enc), past)
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/finish?session_id="+ch.ID, strings.NewReader(`{}`))
		r = withUserID(r, user.ID)
		w := httptest.NewRecorder()
		h.HandleFinishRegistration(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("session owned by different user returns 400", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)
		owner, err := database.CreateUser(t.Context(), "Nina", "nina@example.com", "hash")
		require.NoError(t, err)
		other, err := database.CreateUser(t.Context(), "Oscar", "oscar@example.com", "hash2")
		require.NoError(t, err)

		uid := owner.ID
		future := time.Now().UTC().Add(5 * time.Minute)
		data := passkeyChallengeData{SessionData: webauthn.SessionData{}, Name: "key"}
		enc, err := json.Marshal(data)
		require.NoError(t, err)
		ch, err := database.CreatePasskeyChallenge(t.Context(), &uid, string(enc), future)
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/register/finish?session_id="+ch.ID, strings.NewReader(`{}`))
		r = withUserID(r, other.ID)
		w := httptest.NewRecorder()
		h.HandleFinishRegistration(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPasskeyHandler_FinishAuthentication_InvalidSession(t *testing.T) {
	t.Run("missing session_id returns 400", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		h.HandleFinishAuthentication(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unknown session_id returns 401", func(t *testing.T) {
		h, _ := newTestPasskeyHandler(t)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish?session_id=unknown", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		h.HandleFinishAuthentication(w, r)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("expired session returns 401", func(t *testing.T) {
		h, database := newTestPasskeyHandler(t)

		past := time.Now().UTC().Add(-time.Minute)
		data := passkeyChallengeData{SessionData: webauthn.SessionData{}}
		enc, err := json.Marshal(data)
		require.NoError(t, err)
		ch, err := database.CreatePasskeyChallenge(t.Context(), nil, string(enc), past)
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish?session_id="+ch.ID, strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		h.HandleFinishAuthentication(w, r)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
