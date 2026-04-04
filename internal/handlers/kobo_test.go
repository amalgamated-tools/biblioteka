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

// ---- kobo.DownloadURLs helper ----

func TestKoboDownloadURLs_FiltersUnsupportedFormats(t *testing.T) {
	files := []db.BookFile{
		{ID: "1", FileType: "epub", FileName: "book.epub", FileSize: 100},
		{ID: "2", FileType: "txt", FileName: "book.txt", FileSize: 50},
		{ID: "3", FileType: "pdf", FileName: "book.pdf", FileSize: 200},
	}
	urls := kobo.DownloadURLs("http://localhost", "mytoken", "book-id", files)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs (epub + pdf), got %d", len(urls))
	}
	formats := make(map[string]bool)
	for _, u := range urls {
		if u.Format == "" {
			t.Error("expected non-empty Format in download URL")
		}
		formats[u.Format] = true
		if u.Url == "" {
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

// ---- kobo.BookMetadata with series ----

func TestKoboBookMetadata_WithSeries(t *testing.T) {
	h, _ := setupKoboHandler(t)

	seriesName := "The Dark Tower"
	s, err := h.DB.CreateSeries(t.Context(), seriesName, nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	book, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	pos := 1.0
	series := []db.BookSeriesEntry{{Series: *s, Position: &pos}}
	meta := kobo.BookMetadata(book, nil, series, nil)

	seriesMeta := meta.Series
	if seriesMeta == nil {
		t.Fatal("expected Series in metadata")
	}
	if seriesMeta.Name != seriesName {
		t.Errorf("Series.Name = %v, want %q", seriesMeta.Name, seriesName)
	}
	if seriesMeta.Number != int(1) {
		t.Errorf("Series.Number = %v, want 1", seriesMeta.Number)
	}
}

// ---- kobo.SyncToken with BooksLastID ----

func TestKoboSyncTokenRoundTrip_WithBooksLastID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := kobo.SyncToken{
		BooksLastModified: now,
		BooksLastID:       "some-book-id",
	}
	encoded := kobo.EncodeSyncToken(tok)
	decoded := kobo.ParseSyncToken(encoded)
	if decoded.BooksLastID != tok.BooksLastID {
		t.Errorf("BooksLastID: got %q, want %q", decoded.BooksLastID, tok.BooksLastID)
	}
}
