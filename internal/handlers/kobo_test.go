package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// ---- Sync token round-trip tests ----

func TestKoboSyncTokenRoundTrip_Zero(t *testing.T) {
	tok := koboSyncToken{}
	encoded := encodeKoboSyncToken(tok)
	decoded := parseKoboSyncToken(encoded)
	if !decoded.BooksLastModified.IsZero() {
		t.Errorf("BooksLastModified: got %v, want zero", decoded.BooksLastModified)
	}
	if !decoded.ReadingStateLastModified.IsZero() {
		t.Errorf("ReadingStateLastModified: got %v, want zero", decoded.ReadingStateLastModified)
	}
}

func TestKoboSyncTokenRoundTrip_NonZero(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := koboSyncToken{
		BooksLastModified:        now,
		ReadingStateLastModified: now.Add(-time.Hour),
	}
	encoded := encodeKoboSyncToken(tok)
	decoded := parseKoboSyncToken(encoded)
	if !decoded.BooksLastModified.Equal(tok.BooksLastModified) {
		t.Errorf("BooksLastModified: got %v, want %v", decoded.BooksLastModified, tok.BooksLastModified)
	}
	if !decoded.ReadingStateLastModified.Equal(tok.ReadingStateLastModified) {
		t.Errorf("ReadingStateLastModified: got %v, want %v", decoded.ReadingStateLastModified, tok.ReadingStateLastModified)
	}
}

func TestParseKoboSyncToken_Empty(t *testing.T) {
	tok := parseKoboSyncToken("")
	if !tok.BooksLastModified.IsZero() || !tok.ReadingStateLastModified.IsZero() {
		t.Error("expected zero values for empty token")
	}
}

func TestParseKoboSyncToken_Garbage(t *testing.T) {
	tok := parseKoboSyncToken("not-base64!!!")
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
	user, err := d.CreateUser(context.Background(), "Kobo User", "kobo@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

// koboDeviceHandler returns an http.Handler that composes the Kobo token auth
// middleware with the handler's sub-mux, matching the production setup.
func koboDeviceHandler(h *KoboHandler) http.Handler {
	checker := &testKoboTokenChecker{db: h.DB}
	return auth.KoboTokenAuthMiddleware(checker)(h)
}

// testKoboTokenChecker adapts the test DB to auth.KoboTokenChecker.
type testKoboTokenChecker struct {
	db *db.DB
}

func (c *testKoboTokenChecker) GetKoboTokenByToken(ctx context.Context, token string) (*auth.KoboTokenResult, error) {
	tokenHash := auth.HashKoboToken(token)
	t, err := c.db.GetKoboTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	return &auth.KoboTokenResult{
		UserID: t.UserID,
	}, nil
}

func TestKoboTokenCreate_Success(t *testing.T) {
	h, userID := setupKoboHandler(t)

	body, _ := json.Marshal(koboTokenCreateRequest{Name: "My Kobo"})
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

func TestHandleCoverImage_DataURL(t *testing.T) {
	h, _ := setupKoboHandler(t)
	cover := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("cover-bytes"))
	book, err := h.DB.CreateBook(context.Background(), "Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &cover)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type = %q, want %q", got, "image/png")
	}
	if body := w.Body.String(); body != "cover-bytes" {
		t.Fatalf("body = %q, want %q", body, "cover-bytes")
	}
}

func TestHandleCoverImage_NonImageDataURL(t *testing.T) {
	h, _ := setupKoboHandler(t)
	cover := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte("<script>alert(1)</script>"))
	book, err := h.DB.CreateBook(context.Background(), "Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &cover)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestKoboTokenCreate_EmptyName(t *testing.T) {
	h, userID := setupKoboHandler(t)

	body, _ := json.Marshal(koboTokenCreateRequest{Name: ""})
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

// ---- Kobo device API tests ----

// createTestKoboToken is a helper that creates a token and returns the token value.
func createTestKoboToken(t *testing.T, h *KoboHandler, userID string) string {
	t.Helper()
	body, _ := json.Marshal(koboTokenCreateRequest{Name: "test"})
	rCreate := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	rCreate = withUserID(rCreate, userID)
	wCreate := httptest.NewRecorder()
	h.HandleKoboTokens(wCreate, rCreate)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create token failed: %s", wCreate.Body.String())
	}
	var tok map[string]any
	_ = json.Unmarshal(wCreate.Body.Bytes(), &tok)
	return tok["token"].(string)
}

func TestHandleKobo_UnknownToken(t *testing.T) {
	h, _ := setupKoboHandler(t)
	handler := koboDeviceHandler(h)

	r := httptest.NewRequest(http.MethodGet, "/kobo/badtoken/v1/initialization", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleKobo_Init(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/initialization", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	if w.Header().Get("x-kobo-apitoken") != "e30=" {
		t.Errorf("x-kobo-apitoken = %q, want e30=", w.Header().Get("x-kobo-apitoken"))
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	resources, ok := resp["Resources"].(map[string]any)
	if !ok {
		t.Fatal("expected Resources object in response")
	}
	if resources["library_sync"] == nil {
		t.Error("expected library_sync in Resources")
	}
}

func TestHandleKobo_Sync_EmptyLibrary(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var results []any
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty sync results for empty library, got %d", len(results))
	}

	// Sync token header must be set
	if w.Header().Get("x-kobo-synctoken") == "" {
		t.Error("expected x-kobo-synctoken header in sync response")
	}
}

func TestHandleKobo_Auth_Stub(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	authBody := `{"UserKey":"testuserkey"}`
	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/auth/device",
		bytes.NewBufferString(authBody))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["AccessToken"] == nil || resp["AccessToken"] == "" {
		t.Error("expected non-empty AccessToken in auth response")
	}
	if resp["UserKey"] != "testuserkey" {
		t.Errorf("UserKey = %v, want testuserkey", resp["UserKey"])
	}
}

// ---- Kobo format mapping ----

func TestKoboFormatForFileType(t *testing.T) {
	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{"epub", "EPUB3", true},
		{"EPUB", "EPUB3", true},
		{"kepub", "KEPUB", true},
		{"mobi", "MOBI", true},
		{"pdf", "PDF", true},
		{"azw3", "AZW3", true},
		{"txt", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := koboFormatForFileType(tc.input)
		if ok != tc.ok || got != tc.want {
			t.Errorf("koboFormatForFileType(%q) = (%q, %v), want (%q, %v)",
				tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

// ---- encodeKoboSyncToken produces valid base64 JSON ----

func TestEncodeKoboSyncToken_IsValidBase64JSON(t *testing.T) {
	tok := koboSyncToken{BooksLastModified: time.Now()}
	encoded := encodeKoboSyncToken(tok)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("not valid base64: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if payload["version"] != "1-0-0" {
		t.Errorf("version = %v, want 1-0-0", payload["version"])
	}
}
