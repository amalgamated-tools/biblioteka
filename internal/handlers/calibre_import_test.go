package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/calibre"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

// minimalCalibreSchema is the minimal subset of the Calibre metadata.db schema
// needed to satisfy calibre.LoadBooks.
const minimalCalibreSchema = `
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT 'Unknown',
    sort TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pubdate TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    series_index REAL NOT NULL DEFAULT 1.0,
    author_sort TEXT,
    isbn TEXT DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    has_cover BOOL DEFAULT 0,
    last_modified TIMESTAMP NOT NULL DEFAULT '2000-01-01 00:00:00+00:00'
);
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    sort TEXT,
    link TEXT NOT NULL DEFAULT ''
);
CREATE TABLE books_authors_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    author INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    UNIQUE(book, author)
);
CREATE TABLE series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    sort TEXT
);
CREATE TABLE books_series_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    series INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    UNIQUE(book, series)
);
CREATE TABLE publishers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    sort TEXT
);
CREATE TABLE books_publishers_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    publisher INTEGER NOT NULL REFERENCES publishers(id) ON DELETE CASCADE,
    UNIQUE(book, publisher)
);
CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER UNIQUE NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    text TEXT NOT NULL
);
CREATE TABLE data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    format TEXT NOT NULL,
    uncompressed_size INTEGER NOT NULL,
    name TEXT NOT NULL,
    UNIQUE(book, format)
);
CREATE TABLE identifiers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    val TEXT NOT NULL,
    UNIQUE(book, type)
);
CREATE TABLE languages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lang_code TEXT NOT NULL COLLATE NOCASE
);
CREATE TABLE books_languages_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    lang_code INTEGER NOT NULL REFERENCES languages(id) ON DELETE CASCADE,
    item_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE(book, lang_code)
);
`

// newTestCalibreFile creates a file-backed SQLite database with the Calibre
// schema. It returns the file path and a *sql.DB for inserting test data.
func newTestCalibreFile(t *testing.T) (path string, rawDB *sql.DB) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "metadata.db")
	rawDB, err := sql.Open("sqlite", path)
	require.NoError(t, err, "open calibre sqlite")
	t.Cleanup(func() { _ = rawDB.Close() })
	_, err = rawDB.ExecContext(t.Context(), minimalCalibreSchema)
	require.NoError(t, err, "create calibre schema")
	// Ensure the file is flushed to disk before we read it.
	require.NoError(t, rawDB.PingContext(t.Context()))
	return path, rawDB
}

// insertCalibreBook adds a book row to the test Calibre database.
func insertCalibreBookRow(t *testing.T, rawDB *sql.DB, title string) int64 {
	t.Helper()
	res, err := rawDB.ExecContext(t.Context(),
		`INSERT INTO books (title, path, pubdate, series_index) VALUES (?, ?, '', 1.0)`,
		title, "Author/"+title+" (1)",
	)
	require.NoError(t, err, "insert calibre book row")
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// insertCalibreIdentifierRow adds an identifier to the test Calibre database.
func insertCalibreIdentifierRow(t *testing.T, rawDB *sql.DB, bookID int64, typ, val string) {
	t.Helper()
	_, err := rawDB.ExecContext(t.Context(),
		`INSERT INTO identifiers (book, type, val) VALUES (?, ?, ?)`,
		bookID, typ, val,
	)
	require.NoError(t, err, "insert calibre identifier row")
}

// makeCalibreRequest builds a multipart POST request with metadata_db plus any
// additional form fields. Pass an empty filePath to omit the file field.
func makeCalibreRequest(t *testing.T, url, filePath string, fields map[string]string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if filePath != "" {
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "read calibre db file")
		fw, err := mw.CreateFormFile("metadata_db", "metadata.db")
		require.NoError(t, err, "create form file")
		_, err = fw.Write(content)
		require.NoError(t, err, "write file content")
	}

	for key, val := range fields {
		require.NoError(t, mw.WriteField(key, val), "write field %q", key)
	}

	require.NoError(t, mw.Close())

	r := httptest.NewRequest(http.MethodPost, url, &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

// --- HandlePreview tests ---------------------------------------------------

func TestHandleCalibrePreview_Empty(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	filePath, _ := newTestCalibreFile(t)

	r := makeCalibreRequest(t, "/api/calibre-import/preview", filePath, nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandlePreview(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var preview calibre.Preview
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &preview))
	require.Equal(t, 0, preview.Total)
	require.Empty(t, preview.Books)
}

func TestHandleCalibrePreview_WithBooks(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	filePath, rawDB := newTestCalibreFile(t)
	insertCalibreBookRow(t, rawDB, "Dune")
	insertCalibreBookRow(t, rawDB, "Foundation")

	r := makeCalibreRequest(t, "/api/calibre-import/preview", filePath, nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandlePreview(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var preview calibre.Preview
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &preview))
	require.Equal(t, 2, preview.Total)
	require.Len(t, preview.Books, 2)
}

func TestHandleCalibrePreview_MissingFile(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	r := makeCalibreRequest(t, "/api/calibre-import/preview", "", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandlePreview(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCalibrePreview_NonAdminForbidden(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	// First user is auto-admin; create a second non-admin user.
	_, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	require.NoError(t, err)
	user, err := d.CreateUser(t.Context(), "User", "u@example.com", "password1")
	require.NoError(t, err)

	filePath, _ := newTestCalibreFile(t)

	r := makeCalibreRequest(t, "/api/calibre-import/preview", filePath, nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandlePreview(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleCalibrePreview_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	r := httptest.NewRequest(http.MethodGet, "/api/calibre-import/preview", nil)
	w := httptest.NewRecorder()

	h.HandlePreview(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- HandleImport tests ----------------------------------------------------

func TestHandleCalibreImport_Basic(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	filePath, rawDB := newTestCalibreFile(t)
	bookID := insertCalibreBookRow(t, rawDB, "Dune")
	insertCalibreIdentifierRow(t, rawDB, bookID, "isbn13", "9780441013593")

	r := makeCalibreRequest(t, "/api/calibre-import/confirm", filePath, nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleImport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var result calibre.ImportResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 0, result.Errors)

	books, err := d.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Dune", books[0].Title)
}

func TestHandleCalibreImport_Idempotent(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	filePath, rawDB := newTestCalibreFile(t)
	bookID := insertCalibreBookRow(t, rawDB, "Foundation")
	insertCalibreIdentifierRow(t, rawDB, bookID, "isbn13", "9780553293357")

	// First import.
	r1 := makeCalibreRequest(t, "/api/calibre-import/confirm", filePath, nil)
	r1 = withUserID(r1, user.ID)
	w1 := httptest.NewRecorder()
	h.HandleImport(w1, r1)
	require.Equal(t, http.StatusOK, w1.Code)

	var result1 calibre.ImportResult
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &result1))
	require.Equal(t, 1, result1.Imported)

	// Second import: same ISBN-13 → book should be skipped.
	r2 := makeCalibreRequest(t, "/api/calibre-import/confirm", filePath, nil)
	r2 = withUserID(r2, user.ID)
	w2 := httptest.NewRecorder()
	h.HandleImport(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)

	var result2 calibre.ImportResult
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &result2))
	require.Equal(t, 0, result2.Imported)
	require.Equal(t, 1, result2.Skipped)

	books, err := d.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
}

func TestHandleCalibreImport_WithLibrary(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	lib, err := d.CreateLibrary(t.Context(), "My Library", "/books", "none", false)
	require.NoError(t, err)

	filePath, rawDB := newTestCalibreFile(t)
	bookID := insertCalibreBookRow(t, rawDB, "Dune")
	insertCalibreIdentifierRow(t, rawDB, bookID, "isbn13", "9780441013593")

	r := makeCalibreRequest(t, "/api/calibre-import/confirm", filePath, map[string]string{
		"library_id": lib.ID,
	})
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleImport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	libBooks, err := d.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err)
	require.Len(t, libBooks, 1)
	require.Equal(t, "Dune", libBooks[0].Title)
}

func TestHandleCalibreImport_InvalidLibraryID(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	filePath, rawDB := newTestCalibreFile(t)
	insertCalibreBookRow(t, rawDB, "Dune")

	r := makeCalibreRequest(t, "/api/calibre-import/confirm", filePath, map[string]string{
		"library_id": "nonexistent-library",
	})
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleImport(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCalibreImport_MissingFile(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "Admin", "u@example.com", "password1")
	require.NoError(t, err)
	require.NoError(t, d.SetAdmin(t.Context(), user.ID, true))

	r := makeCalibreRequest(t, "/api/calibre-import/confirm", "", nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleImport(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCalibreImport_MethodNotAllowed(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	r := httptest.NewRequest(http.MethodGet, "/api/calibre-import/confirm", nil)
	w := httptest.NewRecorder()

	h.HandleImport(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleCalibreImport_NonAdminForbidden(t *testing.T) {
	d := newTestDB(t)
	h := &CalibreImportHandler{DB: d}

	// First user is auto-admin; create a second non-admin user.
	_, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	require.NoError(t, err)
	user, err := d.CreateUser(t.Context(), "User", "u@example.com", "password1")
	require.NoError(t, err)

	filePath, rawDB := newTestCalibreFile(t)
	insertCalibreBookRow(t, rawDB, "Dune")

	r := makeCalibreRequest(t, "/api/calibre-import/confirm", filePath, nil)
	r = withUserID(r, user.ID)
	w := httptest.NewRecorder()

	h.HandleImport(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}
