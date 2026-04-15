package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/calibre"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	// maxCalibreDBSize caps the size of an uploaded Calibre metadata.db file.
	maxCalibreDBSize = 100 << 20 // 100 MB
)

// CalibreImportHandler holds dependencies for the Calibre web-import endpoints.
type CalibreImportHandler struct {
	DB *db.DB
}

// calibrePathRequest is the JSON body for server-path based Calibre imports.
type calibrePathRequest struct {
	Path      string `json:"path"`
	LibraryID string `json:"library_id,omitempty"`
}

// calibreSource bundles the opened Calibre database, a cleanup function, and
// any extra fields extracted from the request (like library_id).
type calibreSource struct {
	db        *calibre.DB
	cleanup   func()
	libraryID string
}

// HandlePreview handles POST /api/calibre-import/preview.
// It accepts either a multipart/form-data upload of a Calibre metadata.db or a
// JSON body with a server-side filesystem path, parses the database, and returns
// a summary of the books that would be imported — without writing anything to
// the Biblioteka database.
func (h *CalibreImportHandler) HandlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	src, err := openCalibreSource(w, r)
	if err != nil {
		return
	}
	defer src.cleanup()

	preview, err := calibre.LoadPreview(r.Context(), src.db)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to load calibre preview",
			slog.Any(otelkeys.Error, err),
		)
		// LoadPreview wraps calibre.LoadBooks errors; these are typically caused
		// by a file that is not a valid Calibre metadata.db (e.g.
		// "no such table: books").
		writeError(r.Context(), w, http.StatusBadRequest, "invalid Calibre database: could not read books")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, preview)
}

// HandleImport handles POST /api/calibre-import/confirm.
// It accepts either a multipart/form-data upload of a Calibre metadata.db (with
// an optional library_id field) or a JSON body with a server-side path and
// optional library_id. It imports book metadata (title, authors, series,
// identifiers) into Biblioteka without creating file records.
func (h *CalibreImportHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	src, err := openCalibreSource(w, r)
	if err != nil {
		return
	}
	defer src.cleanup()

	userID := auth.UserIDFromContext(r.Context())

	result, importErr := calibre.WebImport(r.Context(), h.DB, src.db, calibre.WebImportOptions{
		LibraryID: src.libraryID,
	})
	if importErr != nil {
		slog.ErrorContext(r.Context(), "calibre web import failed",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, importErr),
		)
		switch {
		case errors.Is(importErr, calibre.ErrLibraryNotFound):
			writeError(r.Context(), w, http.StatusBadRequest, importErr.Error())
		case errors.Is(importErr, calibre.ErrLoadCalibreBooks):
			writeError(r.Context(), w, http.StatusBadRequest, "invalid Calibre database: could not read books")
		default:
			writeError(r.Context(), w, http.StatusInternalServerError, "import failed; check server logs")
		}
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionCalibreImported, "calibre_import", "", map[string]any{
		"total":      result.Total,
		"imported":   result.Imported,
		"skipped":    result.Skipped,
		"errors":     result.Errors,
		"library_id": src.libraryID,
	})

	slog.InfoContext(r.Context(), "calibre web import complete",
		slog.String(otelkeys.UserID, userID),
		slog.Int(otelkeys.BookCount, result.Total),
		slog.Int(otelkeys.Imported, result.Imported),
		slog.Int(otelkeys.Skipped, result.Skipped),
		slog.Int(otelkeys.ErrorCount, result.Errors),
	)

	writeJSON(r.Context(), w, http.StatusOK, result)
}

// openCalibreSource dispatches on the request Content-Type to obtain a
// calibre.DB. For multipart/form-data it delegates to openUploadedCalibreDB;
// for application/json it opens the server-side path provided in the body.
// On error it writes the HTTP response and returns a non-nil error.
func openCalibreSource(w http.ResponseWriter, r *http.Request) (*calibreSource, error) {
	ct := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		calibreDB, cleanup, err := openUploadedCalibreDB(w, r)
		if err != nil {
			return nil, err
		}
		libraryID := strings.TrimSpace(r.FormValue("library_id"))
		return &calibreSource{db: calibreDB, cleanup: cleanup, libraryID: libraryID}, nil

	case strings.Contains(ct, "application/json"):
		return openServerPathCalibreDB(w, r)

	default:
		writeError(r.Context(), w, http.StatusUnsupportedMediaType, "Content-Type must be multipart/form-data or application/json")
		return nil, errors.New("unsupported content type")
	}
}

// openServerPathCalibreDB decodes a JSON body with a server-side path, validates
// the path, and opens a calibre.DB on it. On error it writes the HTTP response
// and returns a non-nil error.
func openServerPathCalibreDB(w http.ResponseWriter, r *http.Request) (*calibreSource, error) {
	ctx := r.Context()

	var req calibrePathRequest
	if !decodeJSON(r, w, &req) {
		return nil, errors.New("invalid request body")
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		writeError(ctx, w, http.StatusBadRequest, "path is required")
		return nil, errors.New("empty path")
	}

	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		writeError(ctx, w, http.StatusBadRequest, "path must be absolute")
		return nil, errors.New("relative path")
	}

	calibreDB, err := calibre.Open(path)
	if err != nil {
		slog.WarnContext(ctx, "failed to open calibre db at server path",
			slog.String(otelkeys.Path, path),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusBadRequest, "could not open Calibre database at the specified path")
		return nil, err
	}

	cleanup := func() {
		_ = calibreDB.Close()
	}
	return &calibreSource{
		db:        calibreDB,
		cleanup:   cleanup,
		libraryID: strings.TrimSpace(req.LibraryID),
	}, nil
}

// openUploadedCalibreDB parses the multipart request body, saves the
// metadata_db field to a temporary file, opens a calibre.DB on it, and
// returns the DB together with a cleanup function that closes the DB and
// removes the file. On any error it writes the appropriate HTTP error response
// and returns a non-nil error; the caller should return immediately in that
// case. When err is nil the caller MUST call cleanup().
func openUploadedCalibreDB(w http.ResponseWriter, r *http.Request) (*calibre.DB, func(), error) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, maxCalibreDBSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(ctx, w, http.StatusRequestEntityTooLarge, "file too large: Calibre metadata.db must be under 100 MB")
			return nil, func() {}, err
		}
		writeError(ctx, w, http.StatusBadRequest, "invalid multipart form")
		return nil, func() {}, err
	}

	file, _, err := r.FormFile("metadata_db")
	if err != nil {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
		writeError(ctx, w, http.StatusBadRequest, "metadata_db file is required")
		return nil, func() {}, err
	}

	// Save to a temporary file because calibre.Open requires a filesystem path.
	tmpFile, err := os.CreateTemp("", "calibre-metadata-*.db")
	if err != nil {
		_ = file.Close()
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
		slog.ErrorContext(ctx, "failed to create temp file for calibre db", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to process uploaded file")
		return nil, func() {}, err
	}
	tmpPath := tmpFile.Name()

	_, copyErr := io.Copy(tmpFile, file)
	_ = file.Close()
	if r.MultipartForm != nil {
		_ = r.MultipartForm.RemoveAll()
	}
	closeErr := tmpFile.Close()

	removeTmp := func() {
		if rmErr := os.Remove(tmpPath); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.WarnContext(ctx, "failed to remove temp calibre db",
				slog.String(otelkeys.Path, tmpPath),
				slog.Any(otelkeys.Error, rmErr),
			)
		}
	}

	if copyErr != nil {
		removeTmp()
		slog.ErrorContext(ctx, "failed to write temp calibre db", slog.Any(otelkeys.Error, copyErr))
		writeError(ctx, w, http.StatusInternalServerError, "failed to process uploaded file")
		return nil, func() {}, copyErr
	}
	if closeErr != nil {
		slog.WarnContext(ctx, "failed to close temp calibre db", slog.Any(otelkeys.Error, closeErr))
	}

	calibreDB, err := calibre.Open(tmpPath)
	if err != nil {
		removeTmp()
		writeError(ctx, w, http.StatusBadRequest, "invalid Calibre metadata.db file")
		return nil, func() {}, err
	}

	cleanup := func() {
		_ = calibreDB.Close()
		removeTmp()
	}
	return calibreDB, cleanup, nil
}
