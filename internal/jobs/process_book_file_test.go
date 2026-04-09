package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestProcessBookFile_EPUB3CoverExtractedOnImport(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3-with-cover.epub")
	testPNG := testutils.TinyPNG()

	testutils.MakeTestEPUBWithOptions(t, epubPath, "EPUB3 Cover Book", "Some Author", "urn:isbn:9780000000099", testutils.EPUBOptions{
		Version:        "3.0",
		EPUB3Cover:     true,
		CoverImageData: testPNG,
		CoverImageHref: "images/cover.png",
		CoverMediaType: "image/png",
	})

	epubInfo, err := os.Stat(epubPath)
	require.NoError(t, err, "stat epub")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     epubPath,
		FileName: "epub3-with-cover.epub",
		FileType: "epub",
		FileSize: epubInfo.Size(),
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.NotNil(t, books[0].CoverImageURL, "expected EPUB3 book to have a cover image URL after import")
	require.True(t, strings.HasPrefix(*books[0].CoverImageURL, "data:image/png;base64,"))
}

func TestProcessBookFile_NilDatabase(t *testing.T) {
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	err = ProcessBookFile(t.Context(), nil, ext, nil, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "test.epub",
		FileType: "epub",
	})
	require.Error(t, err, "expected error for nil database")
}

func TestProcessBookFile_NilExtractor(t *testing.T) {
	database := newTestDB(t)

	err := ProcessBookFile(t.Context(), database, nil, nil, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "test.epub",
		FileType: "epub",
	})
	require.Error(t, err, "expected error for nil extractor")
}

func TestProcessBookFile_EmptyPath(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     "",
		FileName: "test.epub",
		FileType: "epub",
	})
	require.Error(t, err, "expected error for empty path")
}

func TestProcessBookFile_WhitespacePath(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     "   ",
		FileName: "test.epub",
		FileType: "epub",
	})
	require.Error(t, err, "expected error for whitespace-only path")
}

func TestProcessBookFile_EmptyFileName(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "",
		FileType: "epub",
	})
	require.Error(t, err, "expected error for empty file name")
}

func TestProcessBookFile_EmptyFileType(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     "/tmp/test.epub",
		FileName: "test.epub",
		FileType: "",
	})
	require.Error(t, err, "expected error for empty file type")
}

func TestProcessBookFile_TitleFromFilename(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	// Use a non-epub file so metadata extraction fails and title comes from filename
	dir := t.TempDir()
	path := filepath.Join(dir, "My Cool Book.pdf")
	// Create an empty file so the path exists (extraction will fail but that's OK)
	err = os.WriteFile(path, []byte("not a real pdf"), 0o644)
	require.NoError(t, err, "write test file")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     path,
		FileName: "My Cool Book.pdf",
		FileType: "pdf",
		FileSize: 14,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	// Extension should be stripped since it matches fileType
	require.Equal(t, "My Cool Book", books[0].Title)
}

func TestProcessBookFile_TitleFromFilename_NoExtensionMatch(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	path := filepath.Join(dir, "noext")
	err = os.WriteFile(path, []byte("content"), 0o644)
	require.NoError(t, err, "write test file")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     path,
		FileName: "noext",
		FileType: "pdf",
		FileSize: 7,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	// No extension to strip, so title is the full filename
	require.Equal(t, "noext", books[0].Title)
}

func TestProcessBookFile_ExistingAuthorReused(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	// Pre-create the author
	_, err = database.CreateAuthor(t.Context(), "F. Scott Fitzgerald", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     epubPath,
		FileName: "gatsby.epub",
		FileType: "epub",
		FileSize: 1024,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// Should have reused the existing author, not created a duplicate
	authors, err := database.ListAuthors(t.Context())
	require.NoError(t, err, "list authors")
	require.Len(t, authors, 1, "expected 1 author (reused)")

	// Verify the book is associated with the existing author
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	bookAuthors, err := database.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, bookAuthors, 1, "expected 1 book author")
	require.Equal(t, "F. Scott Fitzgerald", bookAuthors[0].Name)
}

func TestProcessBookFile_PersistsEmbeddedCover(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "cover.epub")

	testutils.MakeTestEPUBWithOptions(t, epubPath, "Book with Cover", "Author", "some-id-123", testutils.EPUBOptions{
		CoverImageData: testutils.TinyJPEG(),
		CoverImageHref: "images/cover.jpg",
		CoverMediaType: "image/jpeg",
	})
	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     epubPath,
		FileName: "cover.epub",
		FileType: "epub",
		FileSize: 1024,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.NotNil(t, books[0].CoverImageURL, "expected embedded cover data URL")
	require.True(t, strings.HasPrefix(*books[0].CoverImageURL, "data:image/jpeg;base64,"))
}

func TestProcessBookFile_NoAuthorInMetadata(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "noauthor.epub")
	// Empty creator means no author in metadata
	testutils.MakeTestEPUB(t, epubPath, "Anonymous Work", "", "some-id-123")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     epubPath,
		FileName: "noauthor.epub",
		FileType: "epub",
		FileSize: 512,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// No author should be created
	authors, err := database.ListAuthors(t.Context())
	require.NoError(t, err, "list authors")
	require.Len(t, authors, 0)

	// Book should still be created
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.Equal(t, "Anonymous Work", books[0].Title)
}

func TestProcessBookFile_MetadataExtractionFails(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	// Create a file that isn't a valid EPUB — extraction will fail
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.epub")
	err = os.WriteFile(path, []byte("not a valid epub"), 0o644)
	require.NoError(t, err, "write broken.epub")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     path,
		FileName: "broken.epub",
		FileType: "epub",
		FileSize: 16,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// Book should still be created with filename-derived title
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.Equal(t, "broken", books[0].Title)
}

func TestProcessBookFile_ISBN10(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "isbn10.epub")
	testutils.MakeTestEPUB(t, epubPath, "ISBN10 Book", "Author", "isbn:0-306-40615-2")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:     epubPath,
		FileName: "isbn10.epub",
		FileType: "epub",
		FileSize: 1024,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.NotNil(t, books[0].ISBN10)
	require.Equal(t, "0306406152", *books[0].ISBN10)
	require.Nil(t, books[0].ISBN13)
}

func TestProcessBookFile_EnqueuesGoodreadsWithUserID(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	user, err := database.CreateUser(t.Context(), "Test User", "test@example.com", "hashedpass")
	require.NoError(t, err, "create user")

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test-enqueue.epub")
	testutils.MakeTestEPUB(t, epubPath, "Enqueue Test", "Test Author", "urn:isbn:9780000000001")

	enq := &genericMockEnqueuer{}

	err = ProcessBookFile(t.Context(), database, ext, enq, ProcessFilePayload{
		Path:     epubPath,
		FileName: "test-enqueue.epub",
		FileType: "epub",
		FileSize: 1024,
		UserID:   user.ID,
	})
	require.NoError(t, err, "ProcessBookFile()")

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1, "expected exactly one enqueued job")
	require.Equal(t, JobEnrichGoodreads, enq.jobs[0].Name)
}

func TestProcessBookFile_SkipsEnqueueWithoutUserID(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test-no-user.epub")
	testutils.MakeTestEPUB(t, epubPath, "No User Test", "Test Author", "urn:isbn:9780000000002")

	enq := &genericMockEnqueuer{}

	err = ProcessBookFile(t.Context(), database, ext, enq, ProcessFilePayload{
		Path:     epubPath,
		FileName: "test-no-user.epub",
		FileType: "epub",
		FileSize: 1024,
		// UserID intentionally empty
	})
	require.NoError(t, err, "ProcessBookFile()")

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Empty(t, enq.jobs, "expected no enqueued jobs when UserID is empty")
}

func TestProcessBookFile_EnqueueFailureDoesNotFailProcessing(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	user, err := database.CreateUser(t.Context(), "Test User", "fail@example.com", "hashedpass")
	require.NoError(t, err, "create user")

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test-fail-enqueue.epub")
	testutils.MakeTestEPUB(t, epubPath, "Fail Enqueue Test", "Test Author", "urn:isbn:9780000000003")

	enq := &genericMockEnqueuer{err: fmt.Errorf("redis unavailable")}

	err = ProcessBookFile(t.Context(), database, ext, enq, ProcessFilePayload{
		Path:     epubPath,
		FileName: "test-fail-enqueue.epub",
		FileType: "epub",
		FileSize: 1024,
		UserID:   user.ID,
	})
	require.NoError(t, err, "ProcessBookFile should succeed even when enqueue fails")

	// Verify the book was still created
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.Equal(t, "Fail Enqueue Test", books[0].Title)
}

func TestProcessBookFile_OverrideTitle(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "original.epub")
	testutils.MakeTestEPUB(t, epubPath, "Metadata Title", "Some Author", "urn:isbn:9780000000001")

	epubInfo, err := os.Stat(epubPath)
	require.NoError(t, err, "stat epub")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:          epubPath,
		FileName:      "original.epub",
		FileType:      "epub",
		FileSize:      epubInfo.Size(),
		OverrideTitle: "User Supplied Title",
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.Equal(t, "User Supplied Title", books[0].Title, "override title should take precedence over extracted metadata")
}

func TestProcessBookFile_OverrideAuthor(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "book.epub")
	testutils.MakeTestEPUB(t, epubPath, "Some Title", "Embedded Author", "urn:isbn:9780000000002")

	epubInfo, err := os.Stat(epubPath)
	require.NoError(t, err, "stat epub")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:           epubPath,
		FileName:       "book.epub",
		FileType:       "epub",
		FileSize:       epubInfo.Size(),
		OverrideAuthor: "User Supplied Author",
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	bookAuthors, err := database.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, bookAuthors, 1)
	require.Equal(t, "User Supplied Author", bookAuthors[0].Name, "override author should take precedence over extracted metadata")
}

func TestProcessBookFile_OverrideDescription(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	// Use a plain file so extraction fails gracefully and we rely entirely on
	// the override values.
	bookPath := filepath.Join(dir, "book.pdf")
	err = os.WriteFile(bookPath, []byte("not a real pdf"), 0o644)
	require.NoError(t, err, "write test file")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:                bookPath,
		FileName:            "book.pdf",
		FileType:            "pdf",
		FileSize:            14,
		OverrideTitle:       "Override Book",
		OverrideDescription: "A compelling description",
		OverrideLanguage:    "fr",
		OverridePublisher:   "Test Publisher",
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	require.Equal(t, "Override Book", books[0].Title)
	require.NotNil(t, books[0].Description)
	require.Equal(t, "A compelling description", *books[0].Description)
	require.NotNil(t, books[0].Language)
	require.Equal(t, "fr", *books[0].Language)
	require.NotNil(t, books[0].Publisher)
	require.Equal(t, "Test Publisher", *books[0].Publisher)
}

func TestProcessBookFile_OverrideISBN13(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	bookPath := filepath.Join(dir, "book.pdf")
	err = os.WriteFile(bookPath, []byte("not a real pdf"), 0o644)
	require.NoError(t, err, "write test file")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:          bookPath,
		FileName:      "book.pdf",
		FileType:      "pdf",
		FileSize:      14,
		OverrideTitle: "ISBN Override Book",
		OverrideISBN:  "9780000000001",
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.NotNil(t, books[0].ISBN13)
	require.Equal(t, "9780000000001", *books[0].ISBN13)
	require.Nil(t, books[0].ISBN10)
}

func TestProcessBookFile_OverrideISBN10(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	bookPath := filepath.Join(dir, "book.pdf")
	err = os.WriteFile(bookPath, []byte("not a real pdf"), 0o644)
	require.NoError(t, err, "write test file")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:          bookPath,
		FileName:      "book.pdf",
		FileType:      "pdf",
		FileSize:      14,
		OverrideTitle: "ISBN10 Override Book",
		OverrideISBN:  "0306406152",
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.NotNil(t, books[0].ISBN10)
	require.Equal(t, "0306406152", *books[0].ISBN10)
	require.Nil(t, books[0].ISBN13)
}

func TestProcessBookFile_OverrideTitleEmptyStringIsIgnored(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	dir := t.TempDir()
	// Name the file "Embedded Title.epub" so the filename-derived title is
	// "Embedded Title" regardless of whether ExifTool is available.
	epubPath := filepath.Join(dir, "Embedded Title.epub")
	testutils.MakeTestEPUB(t, epubPath, "Embedded Title", "Author", "urn:isbn:9780000000003")

	epubInfo, err := os.Stat(epubPath)
	require.NoError(t, err, "stat epub")

	err = ProcessBookFile(t.Context(), database, ext, nil, ProcessFilePayload{
		Path:          epubPath,
		FileName:      "Embedded Title.epub",
		FileType:      "epub",
		FileSize:      epubInfo.Size(),
		OverrideTitle: "", // empty — should not override
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.Equal(t, "Embedded Title", books[0].Title, "empty override title should not replace extracted metadata")
}
