package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"

	"github.com/stretchr/testify/require"
)

func setupBookHandler(t *testing.T) (*BookHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &BookHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestCreateBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	body := mustMarshal(t, bookRequest{Title: "The Gunslinger"})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "The Gunslinger", dto.Title)
	require.NotNil(t, dto.Authors)
	require.NotNil(t, dto.Series)
	require.NotNil(t, dto.Tags)
	require.NotNil(t, dto.Files)
}

func TestCreateBook_MissingTitle(t *testing.T) {
	h, userID := setupBookHandler(t)

	body := mustMarshal(t, bookRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListBooks_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Len(t, resp.Books, 2)
	require.Equal(t, 2, resp.Total)
	require.Equal(t, 50, resp.Limit)
	require.Equal(t, 0, resp.Offset)
}

func TestListBooks_InvalidLimitOffset_NonInt(t *testing.T) {
	h, userID := setupBookHandler(t)

	// Seed some data
	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	// Provide non-integer limit/offset; handler should fall back to defaults.
	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=abc&offset=xyz", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	require.Len(t, resp.Books, 2)
	require.Equal(t, 2, resp.Total)
	// Invalid values should cause defaults to be used.
	require.Equal(t, 50, resp.Limit)
	require.Equal(t, 0, resp.Offset)
}

func TestListBooks_NegativeLimitOffset(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=-5&offset=-10", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	require.Len(t, resp.Books, 2)
	require.Equal(t, 2, resp.Total)

	// Negative values should be clamped to safe non-negative values.
	require.Greater(t, resp.Limit, 0, "limit should be > 0 after clamping")
	require.GreaterOrEqual(t, resp.Offset, 0, "offset should be >= 0 after clamping")
	require.NotEqual(t, -5, resp.Limit)
	require.NotEqual(t, -10, resp.Offset)
}

func TestListBooks_MaxLimitClamping(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	// Request an absurdly large limit; handler should clamp to a maximum.
	r := httptest.NewRequest(http.MethodGet, "/api/books?limit=999999&offset=0", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")

	require.Len(t, resp.Books, 2)
	require.Equal(t, 2, resp.Total)

	// We don't assert the exact max, only that the requested huge limit was clamped.
	require.NotEqual(t, 999999, resp.Limit)
	require.GreaterOrEqual(t, resp.Limit, len(resp.Books), "limit should be >= number of returned books")
}

func TestListBooks_Search_MatchesTitle(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books?query=Gunslinger", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Len(t, resp.Books, 1)
	require.Equal(t, 1, resp.Total)
	require.True(t, len(resp.Books) > 0)
	require.Equal(t, "The Gunslinger", resp.Books[0].Title)
}

func TestListBooks_Search_NoResults(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books?query=nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Len(t, resp.Books, 0)
	require.Equal(t, 0, resp.Total)
}

func TestListBooks_EmptyQuery_ReturnsAll(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	// Empty query string should behave like no query (list all).
	r := httptest.NewRequest(http.MethodGet, "/api/books?query=", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Len(t, resp.Books, 2)
	require.Equal(t, 2, resp.Total)
}

func TestListBooks_WhitespaceOnlyQuery_ReturnsAll(t *testing.T) {
	h, userID := setupBookHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "A Game of Thrones"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	// Whitespace-only query should behave like no query (list all).
	r := httptest.NewRequest(http.MethodGet, "/api/books?query=%20%20%20", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Len(t, resp.Books, 2)
	require.Equal(t, 2, resp.Total)
}

func TestGetBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetBook_ResponseIncludesRelations(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err)
	author, err := h.DB.CreateAuthor(t.Context(), "Frank Herbert", nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, h.DB.SetBookAuthors(t.Context(), b.ID, []string{author.ID}))
	tag, err := h.DB.CreateTag(t.Context(), "science-fiction")
	require.NoError(t, err)
	require.NoError(t, h.DB.SetBookTags(t.Context(), b.ID, []string{tag.ID}))

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "Dune", dto.Title)
	require.NotNil(t, dto.Authors)
	require.NotNil(t, dto.Series)
	require.NotNil(t, dto.Tags)
	require.NotNil(t, dto.Files)
	require.Len(t, dto.Authors, 1)
	require.Equal(t, "Frank Herbert", dto.Authors[0].Name)
	require.Len(t, dto.Tags, 1)
	require.Equal(t, "science-fiction", dto.Tags[0].Name)
}

func TestGetBook_NotFound(t *testing.T) {
	h, userID := setupBookHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/books/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteBook_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodDelete, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteBook_NonAdminForbidden(t *testing.T) {
	h, _ := setupBookHandler(t)

	nonAdmin, err := h.DB.CreateUser(t.Context(), "Non Admin", "nonadmin@example.com", "password2")
	require.NoError(t, err, "create non-admin user")

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Waste Lands"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodDelete, "/api/books/"+b.ID, nil)
	r = withUserID(r, nonAdmin.ID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "admin access required", resp.Error)
}

func TestDeleteBook_NotFound(t *testing.T) {
	h, userID := setupBookHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/books/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "book not found", resp.Error)
}

func TestBookAuthors_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	a, err := h.DB.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	// Set authors
	body := mustMarshal(t, map[string][]string{"author_ids": {a.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	// Get authors
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/authors", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)

	var authors []authorDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &authors), "unmarshal")
	require.Len(t, authors, 1)
	require.Equal(t, "Stephen King", authors[0].Name)
}

func TestBookSeries_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	pos := 1.0
	body := mustMarshal(t, map[string][]db.BookSeriesInput{
		"entries": {{SeriesID: s.ID, Position: &pos}},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	// Get series
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)

	var entries []bookSeriesEntryDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &entries), "unmarshal")
	require.Len(t, entries, 1)
	require.Equal(t, "The Dark Tower", entries[0].Series.Name)
}

func TestBookFiles_Handler(t *testing.T) {
	h, userID := setupBookHandler(t)

	dir := createTestLibrary(t, h.DB)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")

	// Create a file
	body := mustMarshal(t, map[string]any{
		"file_type": "epub",
		"file_name": "gunslinger.epub",
		"file_size": 1024,
		"file_path": filepath.Join(dir, "gunslinger.epub"),
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	// List files
	r2 := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleBookRoutes(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code)

	var files []bookFileDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &files), "unmarshal")
	require.Len(t, files, 1)
}

func TestCreateBook_EnqueuesGoodreadsJob(t *testing.T) {
	h, userID := setupBookHandler(t)
	enq := &mockEnqueuer{}
	h.Enqueuer = enq

	body := mustMarshal(t, bookRequest{Title: "Project Hail Mary"})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1, "expected exactly one enqueued job")
	require.Equal(t, jobs.JobEnrichGoodreads, enq.jobs[0].Name)

	var payload jobs.EnrichGoodreadsPayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
	require.NotEmpty(t, payload.BookID)
	require.Equal(t, userID, payload.UserID)
}

func TestCreateBook_EnqueueFailureDoesNotFailRequest(t *testing.T) {
	h, userID := setupBookHandler(t)
	enq := &mockEnqueuer{err: errors.New("redis unavailable")}
	h.Enqueuer = enq

	body := mustMarshal(t, bookRequest{Title: "The Martian"})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "book creation should succeed even when enqueue fails")

	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "The Martian", dto.Title)
}

func TestCreateBook_NoEnqueuer(t *testing.T) {
	h, userID := setupBookHandler(t)
	// h.Enqueuer is nil by default

	body := mustMarshal(t, bookRequest{Title: "Dune"})
	r := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "book creation should succeed without enqueuer")
}

func TestUpdateBook_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Original Title"})
	require.NoError(t, err, "create book")

	desc := "A great book"
	body := mustMarshal(t, bookRequest{Title: "Updated Title", Description: &desc})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "Updated Title", dto.Title)
	require.NotNil(t, dto.Description)
	require.Equal(t, "A great book", *dto.Description)
}

func TestUpdateBook_MissingTitle(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Original Title"})
	require.NoError(t, err, "create book")

	body := mustMarshal(t, bookRequest{})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBook_NotFound(t *testing.T) {
	h, userID := setupBookHandler(t)

	body := mustMarshal(t, bookRequest{Title: "Any Title"})
	r := httptest.NewRequest(http.MethodPut, "/api/books/nonexistent-id", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Embedded tags in bookDTO (PR #2325) ----

func TestGetBook_EmbedsEmptyTagsArray(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Tagless Book"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal response")
	require.NotNil(t, dto.Tags, "tags field must be present (not null)")
	require.Len(t, dto.Tags, 0)
}

func TestGetBook_EmbedsTags(t *testing.T) {
	h, userID := setupBookHandler(t)
	ctx := t.Context()

	b, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Tagged Book"})
	require.NoError(t, err, "create book")

	tag1, err := h.DB.CreateTag(ctx, "fiction")
	require.NoError(t, err, "create tag")
	tag2, err := h.DB.CreateTag(ctx, "sci-fi")
	require.NoError(t, err, "create tag")
	require.NoError(t, h.DB.SetBookTags(ctx, b.ID, []string{tag1.ID, tag2.ID}), "set book tags")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal response")
	require.Len(t, dto.Tags, 2)
	tagNames := []string{dto.Tags[0].Name, dto.Tags[1].Name}
	require.ElementsMatch(t, []string{"fiction", "sci-fi"}, tagNames)
}

func TestUpdateBook_EmbedsTags(t *testing.T) {
	h, userID := setupBookHandler(t)
	ctx := t.Context()

	b, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Book With Tags"})
	require.NoError(t, err, "create book")

	tag, err := h.DB.CreateTag(ctx, "mystery")
	require.NoError(t, err, "create tag")
	require.NoError(t, h.DB.SetBookTags(ctx, b.ID, []string{tag.ID}), "set book tags")

	body := mustMarshal(t, bookRequest{Title: "Book With Tags Updated"})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto bookDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal response")
	require.Len(t, dto.Tags, 1)
	require.Equal(t, "mystery", dto.Tags[0].Name)
}

func TestPutBookAuthors_CreatesAuditLog(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err, "create book")
	a, err := h.DB.CreateAuthor(t.Context(), "Frank Herbert", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	body := mustMarshal(t, map[string][]string{"author_ids": {a.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/authors", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")
	require.Len(t, logs, 1)
	require.Equal(t, db.AuditActionBookAuthorsUpdated, logs[0].Action)
	require.Equal(t, "book", logs[0].EntityType)
	require.Equal(t, b.ID, logs[0].EntityID)
	require.NotNil(t, logs[0].UserID)
	require.Equal(t, userID, *logs[0].UserID)
}

func TestPutBookTags_CreatesAuditLog(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Dune"})
	require.NoError(t, err, "create book")
	tag, err := h.DB.CreateTag(t.Context(), "science-fiction")
	require.NoError(t, err, "create tag")

	body := mustMarshal(t, map[string][]string{"tag_ids": {tag.ID}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/tags", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")
	require.Len(t, logs, 1)
	require.Equal(t, db.AuditActionBookTagsUpdated, logs[0].Action)
	require.Equal(t, "book", logs[0].EntityType)
	require.Equal(t, b.ID, logs[0].EntityID)
	require.NotNil(t, logs[0].UserID)
	require.Equal(t, userID, *logs[0].UserID)
}

func TestPutBookSeries_CreatesAuditLog(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	pos := 1.0
	body := mustMarshal(t, map[string][]db.BookSeriesInput{
		"entries": {{SeriesID: s.ID, Position: &pos}},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")
	require.Len(t, logs, 1)
	require.Equal(t, db.AuditActionBookSeriesUpdated, logs[0].Action)
	require.Equal(t, "book", logs[0].EntityType)
	require.Equal(t, b.ID, logs[0].EntityID)
	require.NotNil(t, logs[0].UserID)
	require.Equal(t, userID, *logs[0].UserID)
}
