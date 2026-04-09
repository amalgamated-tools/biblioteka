package handlers

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleDownload handles GET /download/{bookID}/{format}.
func (h *KoboHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/download/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeKoboJSON(w, http.StatusBadRequest, map[string]any{})
		return
	}
	bookID := parts[0]
	format := strings.ToLower(parts[1])

	files, err := h.DB.ListBookFiles(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list book files for kobo download", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	var target *db.BookFile
	for i := range files {
		if strings.ToLower(files[i].FileType) == format {
			target = &files[i]
			break
		}
	}
	if target == nil {
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}

	// Re-validate that the stored file path is still within an allowed library
	// root. This catches pre-existing rows that may reference paths outside
	// library directories, and handles symlink-aware canonicalization.
	allowed, pathErr := isBookFilePathAllowed(r.Context(), h.DB, target.FilePath)
	if pathErr != nil {
		slog.ErrorContext(r.Context(), "failed to validate book file path for kobo download", slog.Any(otelkeys.Error, pathErr))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	if !allowed {
		slog.WarnContext(r.Context(), "kobo download blocked: file path outside library roots",
			slog.String(otelkeys.Path, target.FilePath),
		)
		writeKoboJSON(w, http.StatusForbidden, map[string]any{})
		return
	}

	f, err := os.Open(target.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to open book file for kobo download",
			slog.String(otelkeys.Path, target.FilePath),
			slog.Any(otelkeys.Error, err),
		)
		if errors.Is(err, os.ErrNotExist) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		} else {
			writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		}
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": target.FileName}))

	// Increment the download count (best-effort; a failure here should not
	// prevent the download from completing).
	if incErr := h.DB.IncrementBookFileDownloadCount(r.Context(), target.ID); incErr != nil {
		slog.WarnContext(r.Context(), "failed to increment download count",
			slog.String(otelkeys.BookFileID, target.ID),
			slog.Any(otelkeys.Error, incErr),
		)
	}

	http.ServeContent(w, r, target.FileName, stat.ModTime(), f)
}
