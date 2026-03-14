package handlers

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// setupOPDSHandler creates an OPDSHandler with a test DB populated with sample data.
// It returns the handler, a regular user ID, and the user's plaintext password.
func setupOPDSHandler(t *testing.T) (*OPDSHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	jm := newTestJWT(t)

	password := "testpassword"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	user, err := d.CreateUser("Test User", "user@example.com", string(hash))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	h := &OPDSHandler{DB: d, JWT: jm}
	return h, user.ID, password
}

// withBasicAuth adds HTTP Basic Auth credentials to a request.
func withBasicAuth(r *http.Request, username, password string) *http.Request {
	r.SetBasicAuth(username, password)
	return r
}

// parseOPDSFeed parses an OPDS/Atom feed from a response body.
func parseOPDSFeed(t *testing.T, body string) *opdsFeed {
	t.Helper()
	var feed opdsFeed
	if err := xml.Unmarshal([]byte(body), &feed); err != nil {
		t.Fatalf("parse feed XML: %v\nbody: %s", err, body)
	}
	return &feed
}

// --- Middleware authentication tests ---

func TestOPDSMiddleware_NoAuth(t *testing.T) {
	h, _, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	w := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(h.HandleRoot)).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
	if got := w.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Basic") {
		t.Errorf("WWW-Authenticate = %q, want it to contain 'Basic'", got)
	}
}

func TestOPDSMiddleware_BasicAuthValid(t *testing.T) {
	h, _, password := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	r = withBasicAuth(r, "user@example.com", password)
	w := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(h.HandleRoot)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOPDSMiddleware_BasicAuthWrongPassword(t *testing.T) {
	h, _, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	r = withBasicAuth(r, "user@example.com", "wrongpassword")
	w := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(h.HandleRoot)).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestOPDSMiddleware_BasicAuthUnknownUser(t *testing.T) {
	h, _, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	r = withBasicAuth(r, "nobody@example.com", "password")
	w := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(h.HandleRoot)).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestOPDSMiddleware_JWTBearer(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	token, err := h.JWT.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(h.HandleRoot)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOPDSMiddleware_InvalidJWT_FallsBackToBasicAuthPrompt(t *testing.T) {
	h, _, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	r.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(h.HandleRoot)).ServeHTTP(w, r)

	// Invalid JWT and no basic auth credentials → 401.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- HandleRoot tests ---

func TestHandleOPDSRoot_OK(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleRoot(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "atom+xml") {
		t.Errorf("Content-Type = %q, want it to contain 'atom+xml'", ct)
	}

	feed := parseOPDSFeed(t, w.Body.String())
	if feed.Title != "Biblioteka" {
		t.Errorf("feed title = %q, want %q", feed.Title, "Biblioteka")
	}
	if len(feed.Entries) != 3 {
		t.Errorf("entries count = %d, want 3", len(feed.Entries))
	}
}

func TestHandleOPDSRoot_MethodNotAllowed(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/opds/", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleRoot(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- HandleBooks tests ---

func TestHandleOPDSBooks_Empty(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if feed.Title != "All Books" {
		t.Errorf("feed title = %q, want %q", feed.Title, "All Books")
	}
	if len(feed.Entries) != 0 {
		t.Errorf("entries count = %d, want 0", len(feed.Entries))
	}
}

func TestHandleOPDSBooks_WithBooks(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	desc := "A test book"
	publisher := "Test Publisher"
	lang := "en"
	isbn13 := "9780000000001"
	_, err := h.DB.CreateBook("Book One", &desc, nil, nil, &isbn13, nil, nil, nil, nil, &publisher, &lang, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBooks(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(feed.Entries))
	}
	entry := feed.Entries[0]
	if entry.Title != "Book One" {
		t.Errorf("entry title = %q, want %q", entry.Title, "Book One")
	}
	if entry.Publisher != "Test Publisher" {
		t.Errorf("entry publisher = %q, want %q", entry.Publisher, "Test Publisher")
	}
	if entry.Language != "en" {
		t.Errorf("entry language = %q, want %q", entry.Language, "en")
	}
	if entry.Identifier != "urn:isbn:9780000000001" {
		t.Errorf("entry identifier = %q, want %q", entry.Identifier, "urn:isbn:9780000000001")
	}
}

// --- HandleAuthors tests ---

func TestHandleOPDSAuthors_Empty(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if feed.Title != "Authors" {
		t.Errorf("feed title = %q, want %q", feed.Title, "Authors")
	}
}

func TestHandleOPDSAuthors_WithAuthor(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	author, err := h.DB.CreateAuthor("Jane Doe", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthors(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != author.Name {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, author.Name)
	}
}

// --- HandleAuthorBooks tests ---

func TestHandleOPDSAuthorBooks_NotFound(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthorBooks(w, r, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleOPDSAuthorBooks_WithBooks(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	author, err := h.DB.CreateAuthor("Author Name", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}
	book, err := h.DB.CreateBook("My Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	if err := h.DB.SetBookAuthors(book.ID, []string{author.ID}); err != nil {
		t.Fatalf("set book authors: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/"+author.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAuthorBooks(w, r, author.ID)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != "My Book" {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, "My Book")
	}
}

// --- HandleSeriesList tests ---

func TestHandleOPDSSeriesList_Empty(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if feed.Title != "Series" {
		t.Errorf("feed title = %q, want %q", feed.Title, "Series")
	}
}

func TestHandleOPDSSeriesList_WithSeries(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	s, err := h.DB.CreateSeries("The Series", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != s.Name {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, s.Name)
	}
}

// --- HandleSeriesBooks tests ---

func TestHandleOPDSSeriesBooks_NotFound(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesBooks(w, r, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleOPDSSeriesBooks_WithBooks(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	series, err := h.DB.CreateSeries("My Series", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	book, err := h.DB.CreateBook("Series Book 1", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	pos := 1.0
	if err := h.DB.SetBookSeries(book.ID, []db.BookSeriesInput{{SeriesID: series.ID, Position: &pos}}); err != nil {
		t.Fatalf("set book series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/series/"+series.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesBooks(w, r, series.ID)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(feed.Entries))
	}
}

// --- HandleSearch tests ---

func TestHandleOPDSSearch_MissingQuery(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSearch(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOPDSSearch_NoResults(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=nonexistentterm", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSearch(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 0 {
		t.Errorf("entries count = %d, want 0", len(feed.Entries))
	}
}

func TestHandleOPDSSearch_MatchesByTitle(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	_, err := h.DB.CreateBook("The Hobbit", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	_, err = h.DB.CreateBook("Lord of the Rings", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=Hobbit", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSearch(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	feed := parseOPDSFeed(t, w.Body.String())
	if len(feed.Entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != "The Hobbit" {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, "The Hobbit")
	}
}

// --- HandleDownload tests ---

func TestHandleOPDSDownload_NotFound(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/books/somebook/download/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r, "somebook", "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleOPDSDownload_WrongBook(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	book, err := h.DB.CreateBook("A Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	bf, err := h.DB.CreateBookFile(book.ID, "epub", "test.epub", 100, nil, "/tmp/test.epub")
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/books/wrong-book-id/download/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	// Use a different book ID to trigger the ownership check.
	h.HandleDownload(w, r, "wrong-book-id", bf.ID)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- HandleOPDS dispatch tests ---

func TestHandleOPDS_Dispatch(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	tests := []struct {
		path       string
		wantStatus int
		wantTitle  string
	}{
		{"/opds/", http.StatusOK, "Biblioteka"},
		{"/opds", http.StatusOK, "Biblioteka"},
		{"/opds/books", http.StatusOK, "All Books"},
		{"/opds/authors", http.StatusOK, "Authors"},
		{"/opds/series", http.StatusOK, "Series"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			r = withUserID(r, userID)
			w := httptest.NewRecorder()

			h.HandleOPDS(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("path %s: status = %d, want %d; body: %s", tt.path, w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantTitle != "" {
				feed := parseOPDSFeed(t, w.Body.String())
				if feed.Title != tt.wantTitle {
					t.Errorf("path %s: title = %q, want %q", tt.path, feed.Title, tt.wantTitle)
				}
			}
		})
	}
}

func TestHandleOPDS_NotFound(t *testing.T) {
	h, userID, _ := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/unknown/path", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- fileTypeMIME tests ---

func TestFileTypeMIME(t *testing.T) {
	tests := []struct {
		fileType string
		want     string
	}{
		{"epub", "application/epub+zip"},
		{"EPUB", "application/epub+zip"},
		{"pdf", "application/pdf"},
		{"mobi", "application/x-mobipocket-ebook"},
		{"azw", "application/vnd.amazon.ebook"},
		{"azw3", "application/vnd.amazon.ebook"},
		{"cbz", "application/vnd.comicbook+zip"},
		{"cbr", "application/vnd.comicbook-rar"},
		{"fb2", "application/x-fictionbook+xml"},
		{"unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			got := fileTypeMIME(tt.fileType)
			if got != tt.want {
				t.Errorf("fileTypeMIME(%q) = %q, want %q", tt.fileType, got, tt.want)
			}
		})
	}
}

// --- bookToEntry tests ---

func TestBookToEntry_ISBN13(t *testing.T) {
	isbn13 := "9780000000001"
	isbn10 := "0000000001"
	book := &db.Book{
		ID:     "book-id",
		Title:  "Test Book",
		ISBN13: &isbn13,
		ISBN10: &isbn10,
	}
	entry := bookToEntry(book, nil, nil)
	if entry.Identifier != "urn:isbn:9780000000001" {
		t.Errorf("identifier = %q, want urn:isbn:9780000000001", entry.Identifier)
	}
}

func TestBookToEntry_ISBN10Fallback(t *testing.T) {
	isbn10 := "0000000001"
	book := &db.Book{
		ID:     "book-id",
		Title:  "Test Book",
		ISBN10: &isbn10,
	}
	entry := bookToEntry(book, nil, nil)
	if entry.Identifier != "urn:isbn:0000000001" {
		t.Errorf("identifier = %q, want urn:isbn:0000000001", entry.Identifier)
	}
}

func TestBookToEntry_UpdatedFallsBackToCreated(t *testing.T) {
	created := db.Timestamp{}
	created.Time = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	book := &db.Book{
		ID:        "book-id",
		Title:     "Test Book",
		CreatedAt: created,
		// UpdatedAt is zero
	}
	entry := bookToEntry(book, nil, nil)
	if entry.Updated != "2024-01-01T00:00:00Z" {
		t.Errorf("updated = %q, want 2024-01-01T00:00:00Z", entry.Updated)
	}
}

func TestBookToEntry_AcquisitionLinks(t *testing.T) {
	book := &db.Book{
		ID:    "book-123",
		Title: "Downloadable Book",
	}
	files := []db.BookFile{
		{ID: "file-1", FileType: "epub", FileName: "book.epub"},
		{ID: "file-2", FileType: "pdf", FileName: "book.pdf"},
	}
	entry := bookToEntry(book, nil, files)

	var acquisitionLinks []opdsLink
	for _, l := range entry.Links {
		if l.Rel == opdsRelAcquisition {
			acquisitionLinks = append(acquisitionLinks, l)
		}
	}
	if len(acquisitionLinks) != 2 {
		t.Errorf("acquisition links count = %d, want 2", len(acquisitionLinks))
	}
	if acquisitionLinks[0].Href != "/opds/books/book-123/download/file-1" {
		t.Errorf("link href = %q, want /opds/books/book-123/download/file-1", acquisitionLinks[0].Href)
	}
	if acquisitionLinks[0].Type != "application/epub+zip" {
		t.Errorf("link type = %q, want application/epub+zip", acquisitionLinks[0].Type)
	}
}

// --- auth package ExtractToken test ---

func TestExtractTokenExported(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer mytoken")

	token, reason := auth.ExtractToken(r)
	if token != "mytoken" {
		t.Errorf("token = %q, want %q", token, "mytoken")
	}
	if reason != "" {
		t.Errorf("reason = %q, want empty", reason)
	}
}
