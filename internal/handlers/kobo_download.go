package handlers

import (
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

	f, err := os.Open(target.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to open book file for kobo download",
			slog.String(otelkeys.Path, target.FilePath),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": target.FileName}))
	http.ServeContent(w, r, target.FileName, stat.ModTime(), f)
}
