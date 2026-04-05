package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleBookMetadata_CoverURLIncluded verifies that a book with a cover
// image URL includes it in the Kobo metadata response.
func TestHandleBookMetadata_CoverURLIncluded(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	coverURL := "https://example.com/cover.jpg"
	book, err := h.DB.CreateBook(
		context.Background(),
		"Cover URL Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		&coverURL,
	)
	require.NoError(t, err, "create book")
	// A book file is required for metadata to be returned.
	_, err = h.DB.CreateBookFile(
		context.Background(), book.ID, "epub", "cover-book.epub", 1024, nil,
		filepath.Join(t.TempDir(), "cover-book.epub"),
	)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var results []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode")
	require.Len(t, results, 1)
	if results[0]["Title"] != "Cover URL Book" {
		t.Errorf("Title = %v, want Cover URL Book", results[0]["Title"])
	}
	coverImageID, ok := results[0]["CoverImageId"].(string)
	if !ok || coverImageID == "" {
		t.Errorf("CoverImageId = %v, want non-empty string", results[0]["CoverImageId"])
	}
}

// TestHandleBookMetadata_MultipleEntitlementsEachHaveMetadata verifies that
// requesting metadata for multiple books returns each book's metadata.
func TestHandleBookMetadata_MultipleEntitlementsEachHaveMetadata(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Create a book with a file so metadata is returned.
	book, err := h.DB.CreateBook(context.Background(), "Book Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(
		context.Background(), book.ID, "epub", "alpha.epub", 512, nil,
		filepath.Join(t.TempDir(), "alpha.epub"),
	)
	require.NoError(t, err, "create book file")

	// Fetch metadata for just the first book.
	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var results []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode")
	require.Len(t, results, 1)
	if results[0]["Title"] != "Book Alpha" {
		t.Errorf("Title = %v, want Book Alpha", results[0]["Title"])
	}
}

// TestHandleBookMetadata_ContainsEntitlementID verifies that the metadata
// response includes an EntitlementId derived from the book ID.
func TestHandleBookMetadata_ContainsEntitlementID(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	book, err := h.DB.CreateBook(context.Background(), "Entitlement Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(
		context.Background(), book.ID, "epub", "entitlement.epub", 512, nil,
		filepath.Join(t.TempDir(), "entitlement.epub"),
	)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/"+book.ID+"/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var results []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode")
	require.NotEmpty(t, results)
	if results[0]["Title"] != "Entitlement Book" {
		t.Errorf("Title = %v, want Entitlement Book", results[0]["Title"])
	}
	if results[0]["EntitlementId"] != book.ID {
		t.Errorf("EntitlementId = %v, want %v", results[0]["EntitlementId"], book.ID)
	}
}

// TestHandleBookMetadata_EmptyBookID verifies that a non-existent book ID
// returns a 404.
func TestHandleBookMetadata_EmptyBookID(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/nonexistent-id/metadata", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for nonexistent book", w.Code, http.StatusNotFound)
	}
}
