package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	settingWatchFolderPath      = "watch_folder_path"
	settingWatchFolderLibraryID = "watch_folder_library_id"

	// watchFolderPathValidationTimeout guards against blocking the HTTP handler
	// on slow NFS/SMB mounts.
	watchFolderPathValidationTimeout = 5 * time.Second
)

type watchFolderConfigResponse struct {
	Path      string `json:"path"`
	LibraryID string `json:"library_id"`
}

type setWatchFolderConfigRequest struct {
	Path      string `json:"path"`
	LibraryID string `json:"library_id"`
}

// HandleWatchFolderConfig dispatches GET and PUT requests for /api/config/watch-folder.
//
//	@Summary		Get or update watch folder configuration
//	@Description	GET returns current watch folder config (admin only). PUT updates watch folder config (admin only).
//	@Tags			Config
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	watchFolderConfigResponse
//	@Failure		400	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		403	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/config/watch-folder [get]
//	@Router			/config/watch-folder [put]
func (h *ConfigHandler) HandleWatchFolderConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetWatchFolderConfig(w, r)
	case http.MethodPut:
		h.handleSetWatchFolderConfig(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ConfigHandler) handleGetWatchFolderConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	path := h.getSettingOrEmpty(r.Context(), settingWatchFolderPath)
	libraryID := h.getSettingOrEmpty(r.Context(), settingWatchFolderLibraryID)

	writeJSON(r.Context(), w, http.StatusOK, watchFolderConfigResponse{
		Path:      path,
		LibraryID: libraryID,
	})
}

func (h *ConfigHandler) handleSetWatchFolderConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	var req setWatchFolderConfigRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	path := strings.TrimSpace(req.Path)
	libraryID := strings.TrimSpace(req.LibraryID)

	// If path is empty, clear the watch folder config.
	if path == "" {
		if err := h.DB.SetSettings(r.Context(), []db.Setting{
			{Key: settingWatchFolderPath, Value: ""},
			{Key: settingWatchFolderLibraryID, Value: ""},
		}); err != nil {
			slog.ErrorContext(r.Context(), "failed to clear watch folder configuration", slog.Any(otelkeys.Error, err))
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to save watch folder configuration")
			return
		}

		logAudit(r.Context(), h.DB, userID, db.AuditActionWatchFolderUpdated, "config", "watch_folder", map[string]any{
			"path":       "",
			"library_id": "",
		})

		writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "Watch folder configuration cleared"})
		return
	}

	// Validate path exists and is a directory.
	if err := h.validateWatchFolderPath(r.Context(), path); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate library exists.
	if libraryID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "library_id is required when path is set")
		return
	}

	if _, err := h.DB.GetLibrary(r.Context(), libraryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusBadRequest, "library not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to validate library",
			slog.String(otelkeys.LibraryID, libraryID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to validate library")
		return
	}

	if err := h.DB.SetSettings(r.Context(), []db.Setting{
		{Key: settingWatchFolderPath, Value: path},
		{Key: settingWatchFolderLibraryID, Value: libraryID},
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to save watch folder configuration", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to save watch folder configuration")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionWatchFolderUpdated, "config", "watch_folder", map[string]any{
		"path":       path,
		"library_id": libraryID,
	})

	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "Watch folder configuration saved successfully"})
}

// validateWatchFolderPath checks that path exists and is a directory, with
// a timeout to avoid blocking on slow/hung filesystems.
func (h *ConfigHandler) validateWatchFolderPath(ctx context.Context, path string) error {
	return h.validateWatchFolderPathWith(ctx, path, os.Stat)
}

func (h *ConfigHandler) validateWatchFolderPathWith(ctx context.Context, path string, statFn func(string) (os.FileInfo, error)) error {
	validateCtx, cancel := context.WithTimeout(ctx, watchFolderPathValidationTimeout)
	defer cancel()

	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		info, err := statFn(path)
		if err != nil {
			if os.IsNotExist(err) {
				ch <- result{errors.New("folder not found: " + path)}
				return
			}
			ch <- result{err}
			return
		}
		if !info.IsDir() {
			ch <- result{errors.New("path is not a folder: " + path)}
			return
		}
		ch <- result{nil}
	}()

	select {
	case <-validateCtx.Done():
		return errors.New("path validation timed out")
	case r := <-ch:
		return r.err
	}
}

// getSettingOrEmpty returns the setting value for key, or "" if not found.
func (h *ConfigHandler) getSettingOrEmpty(ctx context.Context, key string) string {
	val, err := h.DB.GetSetting(ctx, key)
	if err != nil {
		return ""
	}
	return val
}
