package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/kobo"
)

// ---- Sync token round-trip tests ----

func TestKoboSyncTokenRoundTrip_Zero(t *testing.T) {
	tok := kobo.SyncToken{}
	encoded := kobo.EncodeSyncToken(tok)
	decoded := kobo.ParseSyncToken(encoded)
	if !decoded.BooksLastModified.IsZero() {
		t.Errorf("BooksLastModified: got %v, want zero", decoded.BooksLastModified)
	}
	if !decoded.ReadingStateLastModified.IsZero() {
		t.Errorf("ReadingStateLastModified: got %v, want zero", decoded.ReadingStateLastModified)
	}
}

func TestKoboSyncTokenRoundTrip_NonZero(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := kobo.SyncToken{
		BooksLastModified:        now,
		ReadingStateLastModified: now.Add(-time.Hour),
	}
	encoded := kobo.EncodeSyncToken(tok)
	decoded := kobo.ParseSyncToken(encoded)
	if !decoded.BooksLastModified.Equal(tok.BooksLastModified) {
		t.Errorf("BooksLastModified: got %v, want %v", decoded.BooksLastModified, tok.BooksLastModified)
	}
	if !decoded.ReadingStateLastModified.Equal(tok.ReadingStateLastModified) {
		t.Errorf("ReadingStateLastModified: got %v, want %v", decoded.ReadingStateLastModified, tok.ReadingStateLastModified)
	}
}

func TestParseKoboSyncToken_Empty(t *testing.T) {
	tok := kobo.ParseSyncToken("")
	if !tok.BooksLastModified.IsZero() || !tok.ReadingStateLastModified.IsZero() {
		t.Error("expected zero values for empty token")
	}
}

func TestParseKoboSyncToken_Garbage(t *testing.T) {
	tok := kobo.ParseSyncToken("not-base64!!!")
	if !tok.BooksLastModified.IsZero() {
		t.Error("expected zero BooksLastModified for garbage token")
	}
}

// ---- Token management API tests ----

func setupKoboHandler(t *testing.T) (*KoboHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &KoboHandler{DB: d}
	h.RegisterRoutes()
	user, err := d.CreateUser(t.Context(), "Kobo User", "kobo@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestKoboTokenCreate_Success(t *testing.T) {
	h, userID := setupKoboHandler(t)

	body := mustMarshal(t, koboTokenCreateRequest{Name: "My Kobo"})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var tok map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tok); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tok["token"] == "" || tok["token"] == nil {
		t.Error("expected non-empty token in response")
	}
	if tok["name"] != "My Kobo" {
		t.Errorf("name = %v, want 'My Kobo'", tok["name"])
	}
}

func TestKoboTokenCreate_EmptyName(t *testing.T) {
	h, userID := setupKoboHandler(t)

	body := mustMarshal(t, koboTokenCreateRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKoboTokenList_Empty(t *testing.T) {
	h, userID := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var list []any
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestKoboTokenDelete_NotFound(t *testing.T) {
	h, userID := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/kobo/tokens/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboToken(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
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

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the token is gone.
	listReq := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens", nil)
	listReq = withUserID(listReq, userID)
	listW := httptest.NewRecorder()
	h.HandleKoboTokens(listW, listReq)

	var tokens []any
	if err := json.Unmarshal(listW.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after delete, got %d", len(tokens))
	}
}

func TestKoboTokenCollection_MethodNotAllowed(t *testing.T) {
	h, userID := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodPatch, "/api/kobo/tokens", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestKoboTokenSingle_MethodNotAllowed(t *testing.T) {
	h, userID := setupKoboHandler(t)
	tokenID := createTestKoboTokenID(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens/"+tokenID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKoboToken(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// createTestKoboTokenID creates a token and returns its database ID (not the raw token value).
func createTestKoboTokenID(t *testing.T, h *KoboHandler, userID string) string {
	t.Helper()
	body := mustMarshal(t, koboTokenCreateRequest{Name: "test"})
	rCreate := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	rCreate = withUserID(rCreate, userID)
	wCreate := httptest.NewRecorder()
	h.HandleKoboTokens(wCreate, rCreate)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create token failed: %s", wCreate.Body.String())
	}
	var tok map[string]any
	if err := json.Unmarshal(wCreate.Body.Bytes(), &tok); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	id, ok := tok["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected non-empty id in token response")
	}
	return id
}
