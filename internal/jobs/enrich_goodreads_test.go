package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/goodreads"

	"github.com/stretchr/testify/require"
)

// mockGoodreadsClient implements GoodreadsSearcher for testing.
type mockGoodreadsClient struct {
	searchResult       []goodreads.BookResult
	searchErr          error
	searchByISBNResult []goodreads.BookResult
	searchByISBNErr    error
	getByASINResult    *goodreads.BookResult
	getByASINErr       error
	getByIDResult      *goodreads.BookResult
	getByIDErr         error

	// Track which methods were called
	searchCalled       bool
	searchByISBNCalled bool
	getByASINCalled    bool
	getByIDCalled      bool
}

func (m *mockGoodreadsClient) Search(_ context.Context, _ string) ([]goodreads.BookResult, error) {
	m.searchCalled = true
	return m.searchResult, m.searchErr
}

func (m *mockGoodreadsClient) SearchByISBN(_ context.Context, _ string) ([]goodreads.BookResult, error) {
	m.searchByISBNCalled = true
	return m.searchByISBNResult, m.searchByISBNErr
}

func (m *mockGoodreadsClient) GetBookByASIN(_ context.Context, _ string) (*goodreads.BookResult, error) {
	m.getByASINCalled = true
	return m.getByASINResult, m.getByASINErr
}

func (m *mockGoodreadsClient) GetBookByID(_ context.Context, _ string) (*goodreads.BookResult, error) {
	m.getByIDCalled = true
	return m.getByIDResult, m.getByIDErr
}

var sampleBookResult = goodreads.BookResult{
	WorkID:                "kca://work/amzn1.gr.work.v1.abc123",
	WorkLegacyID:          79106958,
	BookID:                "kca://book/amzn1.gr.book.v3.xyz789",
	BookLegacyID:          54493401,
	BookImageURL:          "https://images-na.ssl-images-amazon.com/images/cover.jpg",
	BookTitle:             "Project Hail Mary",
	BookASIN:              "B08FHBV4ZX",
	BookISBN:              "0593135202",
	BookISBN13:            "9780593135204",
	BookLanguage:          "English",
	AuthorName:            "Andy Weir",
	AuthorID:              "kca://author/amzn1.gr.author.v1.abc",
	AuthorLegacyID:        6540057,
	AuthorProfileImageURL: "https://images.gr-assets.com/authors/andy.jpg",
}

func createTestBookWithFields(t *testing.T, database *db.DB, title string, isbn13, isbn10, asin, grID *string) *db.Book {
	t.Helper()
	book, err := database.CreateBook(t.Context(), title, nil, asin, isbn10, isbn13, grID, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create test book")
	return book
}

func createTestUser(t *testing.T, database *db.DB) *db.User {
	t.Helper()
	user, err := database.CreateUser(t.Context(), "Test User", "test@example.com", "hashedpass")
	require.NoError(t, err, "create test user")
	return user
}

func TestEnrichGoodreads_ISBNLookup(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	isbn13 := "9780593135204"
	book := createTestBookWithFields(t, database, "Project Hail Mary", &isbn13, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchByISBNResult: []goodreads.BookResult{sampleBookResult},
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	require.True(t, mock.searchByISBNCalled, "expected SearchByISBN to be called")
	require.False(t, mock.getByASINCalled, "ASIN lookup should not be called when ISBN succeeds")
	require.False(t, mock.searchCalled, "title search should not be called when ISBN succeeds")

	// Verify the metadata record was created
	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
	require.Equal(t, db.GoodreadsMetadataStatusPending, metadata[0].Status)
	require.NotNil(t, metadata[0].BookID)
	require.Equal(t, book.ID, *metadata[0].BookID)
	require.NotNil(t, metadata[0].Title)
	require.Equal(t, "Project Hail Mary", *metadata[0].Title)
	require.NotNil(t, metadata[0].AuthorName)
	require.Equal(t, "Andy Weir", *metadata[0].AuthorName)
}

func TestEnrichGoodreads_ISBN10Lookup(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	isbn10 := "0593135202"
	book := createTestBookWithFields(t, database, "Project Hail Mary", nil, &isbn10, nil, nil)

	mock := &mockGoodreadsClient{
		searchByISBNResult: []goodreads.BookResult{sampleBookResult},
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)
	require.True(t, mock.searchByISBNCalled)

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
}

func TestEnrichGoodreads_ASINLookup(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	asin := "B08FHBV4ZX"
	book := createTestBookWithFields(t, database, "Project Hail Mary", nil, nil, &asin, nil)

	mock := &mockGoodreadsClient{
		getByASINResult: &sampleBookResult,
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	require.False(t, mock.searchByISBNCalled, "ISBN lookup should not be called when book has no ISBN")
	require.True(t, mock.getByASINCalled, "expected GetBookByASIN to be called")
	require.False(t, mock.searchCalled, "title search should not be called when ASIN succeeds")

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
	require.NotNil(t, metadata[0].ASIN)
	require.Equal(t, "B08FHBV4ZX", *metadata[0].ASIN)
}

func TestEnrichGoodreads_GoodreadsIDLookup(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	grID := "kca://book/amzn1.gr.book.v3.xyz789"
	book := createTestBookWithFields(t, database, "Project Hail Mary", nil, nil, nil, &grID)

	mock := &mockGoodreadsClient{
		getByIDResult: &sampleBookResult,
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	require.True(t, mock.getByIDCalled, "expected GetBookByID to be called")
	require.False(t, mock.searchCalled, "title search should not be called when Goodreads ID succeeds")

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
}

func TestEnrichGoodreads_TitleSearchFallback(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	book := createTestBookWithFields(t, database, "Project Hail Mary", nil, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchResult: []goodreads.BookResult{sampleBookResult},
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	require.False(t, mock.searchByISBNCalled, "ISBN lookup should not be called when book has no ISBN")
	require.False(t, mock.getByASINCalled, "ASIN lookup should not be called when book has no ASIN")
	require.False(t, mock.getByIDCalled, "ID lookup should not be called when book has no Goodreads ID")
	require.True(t, mock.searchCalled, "expected title search to be called as last resort")

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
}

func TestEnrichGoodreads_ISBNPreferredOverTitle(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	isbn13 := "9780593135204"
	book := createTestBookWithFields(t, database, "Project Hail Mary", &isbn13, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchByISBNResult: []goodreads.BookResult{sampleBookResult},
		searchResult:       []goodreads.BookResult{sampleBookResult},
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	require.True(t, mock.searchByISBNCalled, "ISBN lookup should be called first")
	require.False(t, mock.searchCalled, "title search should not be called when ISBN succeeds")
}

func TestEnrichGoodreads_NoMatch(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	book := createTestBookWithFields(t, database, "Obscure Unknown Book", nil, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchResult: nil,
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err, "no match should not return an error")

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Empty(t, metadata, "no metadata record should be created when no match is found")
}

func TestEnrichGoodreads_ISBNFailsFallsToTitle(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	isbn13 := "9780593135204"
	book := createTestBookWithFields(t, database, "Project Hail Mary", &isbn13, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchByISBNErr: fmt.Errorf("network error"),
		searchResult:    []goodreads.BookResult{sampleBookResult},
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	require.True(t, mock.searchByISBNCalled, "ISBN lookup should be attempted")
	require.True(t, mock.searchCalled, "title search should be called when ISBN fails")

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
}

func TestEnrichGoodreads_MissingBookID(t *testing.T) {
	database := newTestDB(t)
	mock := &mockGoodreadsClient{}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: "",
		UserID: "some-user",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "book_id and user_id are required")
}

func TestEnrichGoodreads_MissingUserID(t *testing.T) {
	database := newTestDB(t)
	mock := &mockGoodreadsClient{}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: "some-book",
		UserID: "",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "book_id and user_id are required")
}

func TestEnrichGoodreads_BookNotFound(t *testing.T) {
	database := newTestDB(t)
	mock := &mockGoodreadsClient{}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: "nonexistent-book-id",
		UserID: "some-user",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch book")
}

func TestEnrichGoodreadsHandler_UnmarshalPayload(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	book := createTestBookWithFields(t, database, "Test Book", nil, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchResult: []goodreads.BookResult{sampleBookResult},
	}

	handler := NewEnrichGoodreadsHandler(database, mock)

	payload, err := json.Marshal(EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	err = handler(t.Context(), payload)
	require.NoError(t, err)

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)
}

func TestEnrichGoodreadsHandler_InvalidPayload(t *testing.T) {
	database := newTestDB(t)
	mock := &mockGoodreadsClient{}

	handler := NewEnrichGoodreadsHandler(database, mock)
	err := handler(t.Context(), []byte("not json"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal")
}

func TestEnrichGoodreads_MetadataFieldsMapping(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database)
	book := createTestBookWithFields(t, database, "Test Book", nil, nil, nil, nil)

	mock := &mockGoodreadsClient{
		searchResult: []goodreads.BookResult{sampleBookResult},
	}

	err := enrichGoodreads(t.Context(), database, mock, EnrichGoodreadsPayload{
		BookID: book.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	metadata, err := database.ListGoodreadsMetadataByUser(t.Context(), user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, metadata, 1)

	gm := metadata[0]
	require.Equal(t, db.GoodreadsMetadataStatusPending, gm.Status)
	require.NotNil(t, gm.BookID)
	require.Equal(t, book.ID, *gm.BookID)
	require.NotNil(t, gm.Title)
	require.Equal(t, "Project Hail Mary", *gm.Title)
	require.NotNil(t, gm.ASIN)
	require.Equal(t, "B08FHBV4ZX", *gm.ASIN)
	require.NotNil(t, gm.ISBN10)
	require.Equal(t, "0593135202", *gm.ISBN10)
	require.NotNil(t, gm.ISBN13)
	require.Equal(t, "9780593135204", *gm.ISBN13)
	require.NotNil(t, gm.GoodreadsID)
	require.Equal(t, "kca://book/amzn1.gr.book.v3.xyz789", *gm.GoodreadsID)
	require.NotNil(t, gm.Language)
	require.Equal(t, "English", *gm.Language)
	require.NotNil(t, gm.CoverImageURL)
	require.Equal(t, "https://images-na.ssl-images-amazon.com/images/cover.jpg", *gm.CoverImageURL)
	require.NotNil(t, gm.AuthorName)
	require.Equal(t, "Andy Weir", *gm.AuthorName)
	require.NotNil(t, gm.AuthorGoodreadsID)
	require.Equal(t, "kca://author/amzn1.gr.author.v1.abc", *gm.AuthorGoodreadsID)
	require.NotNil(t, gm.AuthorImageURL)
	require.Equal(t, "https://images.gr-assets.com/authors/andy.jpg", *gm.AuthorImageURL)
	require.NotNil(t, gm.GoodreadsWorkID)
	require.Equal(t, "kca://work/amzn1.gr.work.v1.abc123", *gm.GoodreadsWorkID)
	require.NotNil(t, gm.GoodreadsBookLegacyID)
	require.Equal(t, int64(54493401), *gm.GoodreadsBookLegacyID)
	require.NotNil(t, gm.GoodreadsWorkLegacyID)
	require.Equal(t, int64(79106958), *gm.GoodreadsWorkLegacyID)
	require.NotNil(t, gm.GoodreadsAuthorLegacyID)
	require.Equal(t, int64(6540057), *gm.GoodreadsAuthorLegacyID)
}
