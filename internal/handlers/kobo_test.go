package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
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
		return nil, fmt.Errorf("get token by hash: %w", err)
	}
	return &auth.KoboTokenResult{
		UserID: t.UserID,
	}, nil
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

func TestHandleCoverImage_DataURL(t *testing.T) {
	h, _ := setupKoboHandler(t)
	pngBytes := testutils.TinyPNG()
	cover := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(context.Background(), "Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &cover)
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
	if body := w.Body.Bytes(); !bytes.Equal(body, pngBytes) {
		t.Fatalf("body length = %d, want %d", len(body), len(pngBytes))
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

// ---- Kobo device API tests ----

// createTestKoboToken is a helper that creates a token and returns the token value.
func createTestKoboToken(t *testing.T, h *KoboHandler, userID string) string {
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

// ---- koboRandomUUID ----

func TestKoboRandomUUID_Format(t *testing.T) {
	uuid, err := koboRandomUUID()
	if err != nil {
		t.Fatalf("koboRandomUUID: %v", err)
	}
	// UUID v4 format: 8-4-4-4-12 hex chars
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !re.MatchString(uuid) {
		t.Errorf("UUID %q does not match v4 pattern", uuid)
	}
}

// ---- HandleCoverImage edge cases ----

func TestHandleCoverImage_BookNotFound(t *testing.T) {
	h, _ := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/covers/nonexistent/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCoverImage_NoCover(t *testing.T) {
	h, _ := setupKoboHandler(t)
	book, err := h.DB.CreateBook(context.Background(), "No Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCoverImage_ExternalURL(t *testing.T) {
	h, _ := setupKoboHandler(t)
	externalURL := "https://example.com/cover.jpg"
	book, err := h.DB.CreateBook(context.Background(), "External Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &externalURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}
	if loc := w.Header().Get("Location"); loc != externalURL {
		t.Errorf("Location = %q, want %q", loc, externalURL)
	}
}

func TestHandleCoverImage_EmptyBookID(t *testing.T) {
	h, _ := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/covers/", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ---- HandleDownload ----

func TestHandleDownload_MissingSegments(t *testing.T) {
	h, _ := setupKoboHandler(t)

	// Only one segment (no format)
	r := httptest.NewRequest(http.MethodGet, "/download/onlyone", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDownload_FormatNotFound(t *testing.T) {
	h, _ := setupKoboHandler(t)
	book, err := h.DB.CreateBook(context.Background(), "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	// Book has no files at all, so format won't match.
	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDownload_FileNotFoundOnDisk(t *testing.T) {
	h, _ := setupKoboHandler(t)
	book, err := h.DB.CreateBook(context.Background(), "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	// Register a file in the DB that doesn't exist on disk.
	_, err = h.DB.CreateBookFile(context.Background(), book.ID, "epub", "test.epub", 1024, nil, filepath.Join(t.TempDir(), "nonexistent-kobo-test-file.epub"))
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDownload_Success(t *testing.T) {
	h, _ := setupKoboHandler(t)

	// Write a temp file to serve.
	f, err := os.CreateTemp(t.TempDir(), "test-*.epub")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	content := []byte("fake epub content")
	if _, err := f.Write(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	book, err := h.DB.CreateBook(context.Background(), "Download Test", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	_, err = h.DB.CreateBookFile(context.Background(), book.ID, "epub", "test.epub", int64(len(content)), nil, f.Name())
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !bytes.Equal(w.Body.Bytes(), content) {
		t.Errorf("body = %q, want %q", w.Body.Bytes(), content)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "test.epub") {
		t.Errorf("Content-Disposition = %q, want filename=test.epub", cd)
	}
}

// ---- HandleBookMetadata (via device handler) ----

func TestHandleKobo_BookMetadata_NotFound(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/nonexistent-book-id/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestHandleKobo_BookMetadata_Success(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "Metadata Test", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	_, err = h.DB.CreateBookFile(context.Background(), book.ID, "epub", "book.epub", 2048, nil, filepath.Join(t.TempDir(), "metadata-test.epub"))
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var results []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 metadata result, got %d", len(results))
	}
	if results[0]["Title"] != "Metadata Test" {
		t.Errorf("Title = %v, want 'Metadata Test'", results[0]["Title"])
	}
	urls, _ := results[0]["DownloadUrls"].([]any)
	if len(urls) == 0 {
		t.Error("expected at least one download URL")
	}
}

func TestHandleKobo_BookMetadata_NonGET(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/library/some-id/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	// Non-GET returns 200 with empty array per protocol spec.
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var metadata []any
	if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, w.Body.String())
	}
	if len(metadata) != 0 {
		t.Fatalf("expected empty JSON array, got %d items; body: %s", len(metadata), w.Body.String())
	}
}

// ---- HandleBookState (via device handler) ----

func TestHandleKobo_BookState_GetDefault(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "State Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var states []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &states); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	statusInfo, ok := states[0]["StatusInfo"].(map[string]any)
	if !ok {
		t.Fatalf("StatusInfo is not a map: %v", states[0]["StatusInfo"])
	}
	if statusInfo["Status"] != "ReadyToRead" {
		t.Errorf("Status = %v, want 'ReadyToRead'", statusInfo["Status"])
	}
}

func TestHandleKobo_BookState_GetBookNotFound(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/nonexistent/state", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleKobo_BookState_Update_Success(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "Reading Progress", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	progress := 42.5
	updateBody := map[string]any{
		"ReadingStates": []any{
			map[string]any{
				"StatusInfo": map[string]any{"Status": "Reading"},
				"CurrentBookmark": map[string]any{
					"ProgressPercent": progress,
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	r := httptest.NewRequest(http.MethodPut, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", bytes.NewReader(bodyBytes))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["RequestResult"] != "Success" {
		t.Errorf("RequestResult = %v, want Success", resp["RequestResult"])
	}

	// Verify the state was persisted by fetching it back.
	getR := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", nil)
	getR.Host = "localhost:8080"
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getR)

	var states []map[string]any
	if err := json.Unmarshal(getW.Body.Bytes(), &states); err != nil || len(states) == 0 {
		t.Fatalf("re-fetch state failed: err=%v, count=%d", err, len(states))
	}
	bm, ok := states[0]["CurrentBookmark"].(map[string]any)
	if !ok {
		t.Fatalf("CurrentBookmark is not a map: %v", states[0]["CurrentBookmark"])
	}
	if bm["ProgressPercent"] != 42.5 {
		t.Errorf("persisted ProgressPercent = %v, want 42.5", bm["ProgressPercent"])
	}
}

func TestHandleKobo_BookState_Update_BadRequest(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "Bad State Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	// Send empty ReadingStates array (invalid).
	badBody := `{"ReadingStates":[]}`
	r := httptest.NewRequest(http.MethodPut, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state",
		strings.NewReader(badBody))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleKobo_BookState_Update_BookNotFound(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	updateBody := `{"ReadingStates":[{"StatusInfo":{"Status":"Reading"}}]}`
	r := httptest.NewRequest(http.MethodPut, "/kobo/"+tokenValue+"/v1/library/nonexistent/state",
		strings.NewReader(updateBody))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleKobo_BookState_GetExisting(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "Existing State", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	pct := 75.0
	if _, err := h.DB.UpsertKoboReadingState(context.Background(), userID, book.ID, "Finished", &pct, nil, nil, nil); err != nil {
		t.Fatalf("upsert reading state: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var states []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &states); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	statusInfo, ok := states[0]["StatusInfo"].(map[string]any)
	if !ok {
		t.Fatalf("StatusInfo is not a map: %v", states[0]["StatusInfo"])
	}
	if statusInfo["Status"] != "Finished" {
		t.Errorf("Status = %v, want 'Finished'", statusInfo["Status"])
	}
	bookmark, ok := states[0]["CurrentBookmark"].(map[string]any)
	if !ok {
		t.Fatalf("CurrentBookmark is not a map: %v", states[0]["CurrentBookmark"])
	}
	if bookmark["ProgressPercent"] != 75.0 {
		t.Errorf("ProgressPercent = %v, want 75.0", bookmark["ProgressPercent"])
	}
}

// ---- HandleSync with downloadable books ----

func TestHandleKobo_Sync_WithBooks(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Create a book with a downloadable file so it appears in sync results.
	book, err := h.DB.CreateBook(context.Background(), "Sync Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	_, err = h.DB.CreateBookFile(context.Background(), book.ID, "epub", "sync.epub", 512, nil, filepath.Join(t.TempDir(), "sync-test.epub"))
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var results []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 sync result, got %d", len(results))
	}
	if results[0]["NewEntitlement"] == nil && results[0]["ChangedEntitlement"] == nil {
		t.Error("expected NewEntitlement or ChangedEntitlement in sync result")
	}

	// Extract the entitlement and verify it references the correct book.
	var entitlement map[string]any
	if ne, ok := results[0]["NewEntitlement"].(map[string]any); ok {
		entitlement = ne
	} else if ce, ok := results[0]["ChangedEntitlement"].(map[string]any); ok {
		entitlement = ce
	}
	if entitlement == nil {
		t.Fatal("no entitlement in sync result")
	}
	bm, ok := entitlement["BookMetadata"].(map[string]any)
	if !ok {
		t.Fatal("expected BookMetadata in entitlement")
	}
	if bm["RevisionId"] != book.ID {
		t.Errorf("BookMetadata.RevisionId = %v, want %v", bm["RevisionId"], book.ID)
	}
}

func TestHandleKobo_Sync_SkipsBookWithoutFiles(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Book with no downloadable files should be skipped.
	if _, err := h.DB.CreateBook(context.Background(), "No Files Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var results []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 sync results for book without files, got %d", len(results))
	}
	// Sync token must still be present.
	if w.Header().Get("x-kobo-synctoken") == "" {
		t.Error("expected x-kobo-synctoken header even when no books returned")
	}
}

func TestHandleKobo_Sync_WithReadingState(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "Read Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	_, err = h.DB.CreateBookFile(context.Background(), book.ID, "epub", "read.epub", 512, nil, filepath.Join(t.TempDir(), "read-test.epub"))
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}
	pct := 50.0
	if _, err := h.DB.UpsertKoboReadingState(context.Background(), userID, book.ID, "Reading", &pct, nil, nil, nil); err != nil {
		t.Fatalf("upsert reading state: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var results []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 sync result, got %d", len(results))
	}

	// The entitlement should include the reading state.
	var entitlement map[string]any
	if ne, ok := results[0]["NewEntitlement"].(map[string]any); ok {
		entitlement = ne
	} else if ce, ok := results[0]["ChangedEntitlement"].(map[string]any); ok {
		entitlement = ce
	}
	if entitlement == nil {
		t.Fatal("no entitlement in sync result")
	}
	if entitlement["ReadingState"] == nil {
		t.Fatal("expected ReadingState in sync result for book with reading state")
	}
	rs, ok := entitlement["ReadingState"].(map[string]any)
	if !ok {
		t.Fatal("ReadingState is not a map")
	}
	rsStatusInfo, ok := rs["StatusInfo"].(map[string]any)
	if !ok {
		t.Fatal("ReadingState StatusInfo is not a map")
	}
	if rsStatusInfo["Status"] != "Reading" {
		t.Errorf("ReadingState status = %v, want Reading", rsStatusInfo["Status"])
	}
	rsBm, ok := rs["CurrentBookmark"].(map[string]any)
	if !ok {
		t.Fatal("ReadingState CurrentBookmark is not a map")
	}
	if rsBm["ProgressPercent"] != 50.0 {
		t.Errorf("ProgressPercent = %v, want 50.0", rsBm["ProgressPercent"])
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

// ---- Auxiliary Kobo routes ----

func TestHandleKobo_LoyaltyBenefits(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/user/loyalty/benefits", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["Benefits"] == nil {
		t.Error("expected Benefits in loyalty benefits response")
	}
}

func TestHandleKobo_AnalyticsGetTests(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/analytics/gettests", nil)
	r.Host = "localhost:8080"
	r.Header.Set("X-Kobo-userkey", "testuserkey")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["Result"] != "Success" {
		t.Errorf("Result = %v, want Success", resp["Result"])
	}
	if resp["TestKey"] != "testuserkey" {
		t.Errorf("TestKey = %v, want testuserkey", resp["TestKey"])
	}
}

func TestHandleKobo_DefaultRoute(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/unknown/endpoint", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleKobo_LibraryRoute_Unknown(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// A /v1/library/ path that doesn't end in /metadata or /state.
	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/some-uuid/prices", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---- koboDownloadURLs helper ----

func TestKoboDownloadURLs_FiltersUnsupportedFormats(t *testing.T) {
	files := []db.BookFile{
		{ID: "1", FileType: "epub", FileName: "book.epub", FileSize: 100},
		{ID: "2", FileType: "txt", FileName: "book.txt", FileSize: 50},
		{ID: "3", FileType: "pdf", FileName: "book.pdf", FileSize: 200},
	}
	urls := koboDownloadURLs("http://localhost", "mytoken", "book-id", files)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs (epub + pdf), got %d", len(urls))
	}
	formats := make(map[string]bool)
	for _, u := range urls {
		f, _ := u["Format"].(string)
		if f == "" {
			t.Error("expected non-empty Format in download URL")
		}
		formats[f] = true
		if u["Url"] == "" {
			t.Error("expected non-empty Url in download URL")
		}
	}
	if !formats["EPUB3"] {
		t.Error("expected EPUB3 format in download URLs")
	}
	if !formats["PDF"] {
		t.Error("expected PDF format in download URLs")
	}
}

// ---- koboBookMetadata with series ----

func TestKoboBookMetadata_WithSeries(t *testing.T) {
	h, _ := setupKoboHandler(t)

	seriesName := "The Dark Tower"
	s, err := h.DB.CreateSeries(context.Background(), seriesName, nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	book, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	pos := 1.0
	series := []db.BookSeriesEntry{{Series: *s, Position: &pos}}
	meta := koboBookMetadata(book, nil, series, nil)

	seriesMeta, ok := meta["Series"].(map[string]any)
	if !ok {
		t.Fatal("expected Series in metadata")
	}
	if seriesMeta["Name"] != seriesName {
		t.Errorf("Series.Name = %v, want %q", seriesMeta["Name"], seriesName)
	}
	if seriesMeta["Number"] != int(1) {
		t.Errorf("Series.Number = %v, want 1", seriesMeta["Number"])
	}
}

// ---- koboSyncToken with BooksLastID ----

func TestKoboSyncTokenRoundTrip_WithBooksLastID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := koboSyncToken{
		BooksLastModified: now,
		BooksLastID:       "some-book-id",
	}
	encoded := encodeKoboSyncToken(tok)
	decoded := parseKoboSyncToken(encoded)
	if decoded.BooksLastID != tok.BooksLastID {
		t.Errorf("BooksLastID: got %q, want %q", decoded.BooksLastID, tok.BooksLastID)
	}
}
