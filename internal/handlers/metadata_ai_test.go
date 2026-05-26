package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/stretchr/testify/require"
)

// --- AI Fetch ---

func TestFetchAIEnrichment_NoProvider(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	// LLMProvider is nil by default from setupMetadataHandler.
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-fetch", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestFetchAIEnrichment_Enqueues(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	h.LLMProvider = &stubLLMProvider{}
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-fetch", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusAccepted, w.Code)

	var resp fetchMetadataResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "enqueued", resp.Status)
	require.Equal(t, "mock-job-id", resp.TaskID)
}

func TestFetchAIEnrichment_AlreadyExists(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	h.LLMProvider = &stubLLMProvider{}
	book := createTestBook(t, h.DB, "Test Book")

	// Create a pending enrichment.
	_, err := h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"fiction"}, nil, nil, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-fetch", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusAccepted, w.Code)
	var resp fetchMetadataResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "already_exists", resp.Status)
}

func TestFetchAIEnrichment_EnqueueError(t *testing.T) {
	d := newTestDB(t)
	enq := &mockEnqueuer{err: errors.New("redis unavailable")}
	h := &MetadataHandler{DB: d, Enqueuer: enq, LLMProvider: &stubLLMProvider{}}

	user, err := d.CreateUser(t.Context(), "Test User", "dup@example.com", "password1")
	require.NoError(t, err)

	book := createTestBook(t, d, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-fetch", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- AI Get Pending ---

func TestGetPendingAIEnrichment_Found(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	desc := "A great book"
	_, err := h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"fiction"}, nil, &desc, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+book.ID+"/metadata/ai", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	var dto aiEnrichmentDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "pending", dto.Status)
	require.Equal(t, "ollama", dto.Provider)
	require.Contains(t, dto.SuggestedTags, "fiction")
}

func TestGetPendingAIEnrichment_NotFound(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+book.ID+"/metadata/ai", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// --- AI Apply ---

func TestApplyAIEnrichment_TagsAndDescription(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	desc := "AI generated description"
	_, err := h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"sci-fi", "adventure"}, nil, &desc, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	var dto aiEnrichmentDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "applied", dto.Status)

	// Verify tags were created and assigned.
	tags, err := h.DB.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 2)

	// Verify description was set.
	updated, err := h.DB.GetBook(t.Context(), book.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.Description)
	require.Equal(t, "AI generated description", *updated.Description)
}

func TestApplyAIEnrichment_SkipsDescriptionWhenPresent(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)

	existingDesc := "Original description"
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book", Description: &existingDesc})
	require.NoError(t, err)

	aiDesc := "AI description"
	_, err = h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"fiction"}, nil, &aiDesc, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	// Original description should be preserved.
	updated, err := h.DB.GetBook(t.Context(), book.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.Description)
	require.Equal(t, "Original description", *updated.Description)
}

func TestApplyAIEnrichment_NoPending(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestApplyAIEnrichment_UnionMergesTags(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	// Create an existing tag and assign it to the book.
	existingTag, err := h.DB.FindOrCreateTag(t.Context(), "existing-tag")
	require.NoError(t, err)
	require.NoError(t, h.DB.SetBookTags(t.Context(), book.ID, []string{existingTag.ID}))

	// Create enrichment with a new tag and the existing tag.
	_, err = h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"new-tag", "existing-tag"}, nil, nil, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	tags, err := h.DB.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 2, "should have both existing-tag and new-tag")
}

// --- AI Reject ---

func TestRejectAIEnrichment(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	_, err := h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"fiction"}, nil, nil, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-reject", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify no pending enrichment remains.
	rGet := httptest.NewRequest(http.MethodGet, "/api/books/"+book.ID+"/metadata/ai", nil)
	rGet = withUserID(rGet, userID)
	wGet := httptest.NewRecorder()

	h.HandleBookMetadata(wGet, rGet, book.ID)
	require.Equal(t, http.StatusNotFound, wGet.Code)
}

func TestRejectAIEnrichment_NoPending(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-reject", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestApplyAIEnrichment_SkipsBlankTags(t *testing.T) {
	h, _, userID := setupMetadataHandler(t)
	book := createTestBook(t, h.DB, "Test Book")

	// Create enrichment with blank/whitespace-only tags mixed in.
	_, err := h.DB.CreateAIEnrichment(t.Context(), userID, &book.ID, "ollama", "llama3", []string{"valid-tag", "", "  ", "another-valid"}, nil, nil, "{}")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+book.ID+"/metadata/ai-apply", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookMetadata(w, r, book.ID)

	require.Equal(t, http.StatusOK, w.Code)

	tags, err := h.DB.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 2, "should only have the two valid tags, blank ones skipped")
}

// stubLLMProvider is a no-op llm.Provider for handler tests that only need
// a non-nil provider to pass the availability check.
type stubLLMProvider struct{}

func (s *stubLLMProvider) Generate(_ context.Context, _ string) (string, error) {
	return "", nil
}

// Compile-time check.
var _ llm.Provider = (*stubLLMProvider)(nil)
