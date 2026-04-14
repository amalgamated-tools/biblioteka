package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
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

// HandlePreview handles POST /api/calibre-import/preview.
// It accepts a multipart/form-data upload of a Calibre metadata.db, parses
// it, and returns a summary of the books that would be imported — without
// writing anything to the Biblioteka database.
func (h *CalibreImportHandler) HandlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !requireAdmin(h.DB, w, r) {
		return
	}

	calibreDB, cleanup, err := openUploadedCalibreDB(w, r)
	if err != nil {
		return
	}
	defer cleanup()

	preview, err := calibre.LoadPreview(r.Context(), calibreDB)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to load calibre preview",
			slog.Any(otelkeys.Error, err),
		)
		// LoadPreview wraps calibre.LoadBooks errors; these are typically caused
		// by an uploaded file that is not a valid Calibre metadata.db (e.g.
		// "no such table: books").
		writeError(r.Context(), w, http.StatusBadRequest, "invalid Calibre database: could not read books")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, preview)
}

// HandleImport handles POST /api/calibre-import/confirm.
// It accepts a multipart/form-data upload of a Calibre metadata.db plus an
// optional library_id field, and imports book metadata (title, authors,
// series, identifiers) into Biblioteka without creating file records.
func (h *CalibreImportHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !requireAdmin(h.DB, w, r) {
		return
	}

	calibreDB, cleanup, err := openUploadedCalibreDB(w, r)
	if err != nil {
		return
	}
	defer cleanup()

	libraryID := strings.TrimSpace(r.FormValue("library_id"))
	userID := auth.UserIDFromContext(r.Context())

	result, importErr := calibre.WebImport(r.Context(), h.DB, calibreDB, calibre.WebImportOptions{
		LibraryID: libraryID,
	})
	if importErr != nil {
		slog.ErrorContext(r.Context(), "calibre web import failed",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, importErr),
		)
		// WebImport returns validation errors (invalid library, unreadable
		// Calibre DB) that are client-correctable. Internal per-book failures
		// are counted in result.Errors and do not bubble up here.
		if strings.Contains(importErr.Error(), "load calibre books") {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid Calibre database: could not read books")
		} else {
			writeError(r.Context(), w, http.StatusBadRequest, importErr.Error())
		}
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionCalibreImported, "calibre_import", "", map[string]any{
		"total":      result.Total,
		"imported":   result.Imported,
		"skipped":    result.Skipped,
		"errors":     result.Errors,
		"library_id": libraryID,
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
