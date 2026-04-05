package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

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

// ---- Kobo device API tests ----

// createTestKoboToken is a helper that creates a token and returns the token value.
func createTestKoboToken(t *testing.T, h *KoboHandler, userID string) string {
	t.Helper()
	body := mustMarshal(t, koboTokenCreateRequest{Name: "test"})
	rCreate := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	rCreate = withUserID(rCreate, userID)
	wCreate := httptest.NewRecorder()
	h.HandleKoboTokens(wCreate, rCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &tok), "unmarshal token response")
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

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	if w.Header().Get("x-kobo-apitoken") != "e30=" {
		t.Errorf("x-kobo-apitoken = %q, want e30=", w.Header().Get("x-kobo-apitoken"))
	}

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	resources, ok := resp["Resources"].(map[string]any)
	require.True(t, ok)
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

	require.Equal(t, http.StatusOK, w.Code)

	var results []any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results), "unmarshal")
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

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal auth response")
	if resp["AccessToken"] == nil || resp["AccessToken"] == "" {
		t.Error("expected non-empty AccessToken in auth response")
	}
	if resp["UserKey"] != "testuserkey" {
		t.Errorf("UserKey = %v, want testuserkey", resp["UserKey"])
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

	book, err := h.DB.CreateBook(t.Context(), "Metadata Test", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(t.Context(), book.ID, "epub", "book.epub", 2048, nil, filepath.Join(t.TempDir(), "metadata-test.epub"))
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var results []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results), "unmarshal")
	require.Len(t, results, 1)
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
	require.Equal(t, http.StatusOK, w.Code)

	var metadata []any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &metadata), "unmarshal; body: %s", w.Body.String())
	require.Len(t, metadata, 0)
}

// ---- HandleBookState (via device handler) ----

func TestHandleKobo_BookState_GetDefault(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(t.Context(), "State Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var states []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &states), "unmarshal")
	require.Len(t, states, 1)
	statusInfo, ok := states[0]["StatusInfo"].(map[string]any)
	require.True(t, ok)
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

	book, err := h.DB.CreateBook(t.Context(), "Reading Progress", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

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
	require.NoError(t, err, "marshal")

	r := httptest.NewRequest(http.MethodPut, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", bytes.NewReader(bodyBytes))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp["RequestResult"] != "Success" {
		t.Errorf("RequestResult = %v, want Success", resp["RequestResult"])
	}

	// Verify the state was persisted by fetching it back.
	getR := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", nil)
	getR.Host = "localhost:8080"
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getR)

	var states []map[string]any
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &states), "unmarshal states")
	require.NotEmpty(t, states)
	bm, ok := states[0]["CurrentBookmark"].(map[string]any)
	require.True(t, ok)
	if bm["ProgressPercent"] != 42.5 {
		t.Errorf("persisted ProgressPercent = %v, want 42.5", bm["ProgressPercent"])
	}
}

func TestHandleKobo_BookState_Update_BadRequest(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(t.Context(), "Bad State Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

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

	book, err := h.DB.CreateBook(t.Context(), "Existing State", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	pct := 75.0
	if _, err := h.DB.UpsertKoboReadingState(t.Context(), userID, book.ID, "Finished", &pct, nil, nil, nil); err != nil {
		require.NoError(t, err, "upsert reading state")
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/state", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var states []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &states), "unmarshal")
	require.Len(t, states, 1)
	statusInfo, ok := states[0]["StatusInfo"].(map[string]any)
	require.True(t, ok)
	if statusInfo["Status"] != "Finished" {
		t.Errorf("Status = %v, want 'Finished'", statusInfo["Status"])
	}
	bookmark, ok := states[0]["CurrentBookmark"].(map[string]any)
	require.True(t, ok)
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
	book, err := h.DB.CreateBook(t.Context(), "Sync Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(t.Context(), book.ID, "epub", "sync.epub", 512, nil, filepath.Join(t.TempDir(), "sync-test.epub"))
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var results []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results), "unmarshal")
	require.Len(t, results, 1)
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
	require.NotNil(t, entitlement)
	bm, ok := entitlement["BookMetadata"].(map[string]any)
	require.True(t, ok)
	if bm["RevisionId"] != book.ID {
		t.Errorf("BookMetadata.RevisionId = %v, want %v", bm["RevisionId"], book.ID)
	}
}

func TestHandleKobo_Sync_SkipsBookWithoutFiles(t *testing.T) {
	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Book with no downloadable files should be skipped.
	if _, err := h.DB.CreateBook(t.Context(), "No Files Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var results []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results), "unmarshal")
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

	book, err := h.DB.CreateBook(t.Context(), "Read Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(t.Context(), book.ID, "epub", "read.epub", 512, nil, filepath.Join(t.TempDir(), "read-test.epub"))
	require.NoError(t, err, "create book file")
	pct := 50.0
	if _, err := h.DB.UpsertKoboReadingState(t.Context(), userID, book.ID, "Reading", &pct, nil, nil, nil); err != nil {
		require.NoError(t, err, "upsert reading state")
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var results []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results), "unmarshal")
	require.Len(t, results, 1)

	// The entitlement should include the reading state.
	var entitlement map[string]any
	if ne, ok := results[0]["NewEntitlement"].(map[string]any); ok {
		entitlement = ne
	} else if ce, ok := results[0]["ChangedEntitlement"].(map[string]any); ok {
		entitlement = ce
	}
	require.NotNil(t, entitlement)
	require.NotNil(t, entitlement["ReadingState"])
	rs, ok := entitlement["ReadingState"].(map[string]any)
	require.True(t, ok)
	rsStatusInfo, ok := rs["StatusInfo"].(map[string]any)
	require.True(t, ok)
	if rsStatusInfo["Status"] != "Reading" {
		t.Errorf("ReadingState status = %v, want Reading", rsStatusInfo["Status"])
	}
	rsBm, ok := rs["CurrentBookmark"].(map[string]any)
	require.True(t, ok)
	if rsBm["ProgressPercent"] != 50.0 {
		t.Errorf("ProgressPercent = %v, want 50.0", rsBm["ProgressPercent"])
	}
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
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
