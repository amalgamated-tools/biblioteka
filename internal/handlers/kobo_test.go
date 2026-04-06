package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/kobo"

	"github.com/stretchr/testify/require"
)

// ---- Sync token round-trip tests ----

func TestKoboSyncTokenRoundTrip_Zero(t *testing.T) {
	tok := kobo.SyncToken{}
	encoded := kobo.EncodeSyncToken(tok)
	decoded := kobo.ParseSyncToken(encoded)
	require.True(t, decoded.BooksLastModified.IsZero())
	require.True(t, decoded.ReadingStateLastModified.IsZero())
}

func TestKoboSyncTokenRoundTrip_NonZero(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := kobo.SyncToken{
		BooksLastModified:        now,
		ReadingStateLastModified: now.Add(-time.Hour),
	}
	encoded := kobo.EncodeSyncToken(tok)
	decoded := kobo.ParseSyncToken(encoded)
	require.True(t, decoded.BooksLastModified.Equal(tok.BooksLastModified))
	require.True(t, decoded.ReadingStateLastModified.Equal(tok.ReadingStateLastModified))
}

func TestParseKoboSyncToken_Empty(t *testing.T) {
	tok := kobo.ParseSyncToken("")
	require.True(t, tok.BooksLastModified.IsZero())
	require.True(t, tok.ReadingStateLastModified.IsZero())
}

func TestParseKoboSyncToken_Garbage(t *testing.T) {
	tok := kobo.ParseSyncToken("not-base64!!!")
	require.True(t, tok.BooksLastModified.IsZero())
}

// ---- Token management API tests ----

func setupKoboHandler(t *testing.T) (*KoboHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &KoboHandler{DB: d}
	h.RegisterRoutes()
	user, err := d.CreateUser(t.Context(), "Kobo User", "kobo@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestKoboTokenCreate_Success(t *testing.T) {
	h, userID := setupKoboHandler(t)

	body := mustMarshal(t, koboTokenCreateRequest{Name: "My Kobo"})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var tok map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tok), "unmarshal")
	require.NotEmpty(t, tok["token"], "expected non-empty token in response")
	require.Equal(t, "My Kobo", tok["name"])
}

func TestKoboTokenCreate_EmptyName(t *testing.T) {
	h, userID := setupKoboHandler(t)

	body := mustMarshal(t, koboTokenCreateRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestKoboTokenList_Empty(t *testing.T) {
	h, userID := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var list []any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list), "unmarshal")
	require.Len(t, list, 0)
}

func TestKoboTokenDelete_NotFound(t *testing.T) {
	h, userID := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/kobo/tokens/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboToken(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Token management: delete success ----

func TestKoboTokenDelete_Success(t *testing.T) {
	h, userID := setupKoboHandler(t)

	// Create a token to delete.
	tokenID := createTestKoboTokenID(t, h, userID)

	r := httptest.NewRequest(http.MethodDelete, "/api/kobo/tokens/"+tokenID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboToken(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify the token is gone.
	listReq := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens", nil)
	listReq = withUserID(listReq, userID)
	listW := httptest.NewRecorder()
	h.HandleKoboTokens(listW, listReq)

	var tokens []any
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &tokens), "unmarshal")
	require.Len(t, tokens, 0)
}

func TestKoboTokenCollection_MethodNotAllowed(t *testing.T) {
	h, userID := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodPatch, "/api/kobo/tokens", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestKoboTokenSingle_MethodNotAllowed(t *testing.T) {
	h, userID := setupKoboHandler(t)
	tokenID := createTestKoboTokenID(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens/"+tokenID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboToken(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// createTestKoboTokenID creates a token and returns its database ID (not the raw token value).
func createTestKoboTokenID(t *testing.T, h *KoboHandler, userID string) string {
	t.Helper()
	body := mustMarshal(t, koboTokenCreateRequest{Name: "test"})
	rCreate := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	rCreate = withUserID(rCreate, userID)
	wCreate := httptest.NewRecorder()
	h.HandleKoboTokens(wCreate, rCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &tok), "unmarshal")
	id, ok := tok["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, id)
	return id
}
