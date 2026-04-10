package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	settingWatchFolderPath      = db.SettingWatchFolderPath
	settingWatchFolderLibraryID = db.SettingWatchFolderLibraryID

	// watchFolderPathValidationTimeout guards against blocking the HTTP handler
	// on slow NFS/SMB mounts.
	watchFolderPathValidationTimeout = 5 * time.Second

	// maxConcurrentPathValidations limits how many goroutines can be running
	// os.Stat concurrently. This prevents unbounded goroutine accumulation
	// when stat calls hang (e.g., stuck NFS/SMB mounts).
	maxConcurrentPathValidations = 4
)

// pathValidationSem limits concurrent os.Stat goroutines for watch folder
// path validation. Without this, hung filesystem calls could accumulate
// goroutines indefinitely as each timed-out request spawns another.
var pathValidationSem = struct {
	ch   chan struct{}
	once sync.Once
}{}

func acquirePathValidationSlot() chan struct{} {
	pathValidationSem.once.Do(func() {
		pathValidationSem.ch = make(chan struct{}, maxConcurrentPathValidations)
	})
	return pathValidationSem.ch
}

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

	// Validate path is absolute.
	if !filepath.IsAbs(path) {
		writeError(r.Context(), w, http.StatusBadRequest, "watch folder path must be absolute")
		return
	}

	// Validate path exists and is a directory.
	if err := h.validateWatchFolderPath(r.Context(), path); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.WarnContext(r.Context(), "watch folder path validation timed out",
				slog.String(otelkeys.WatchFolderPath, path),
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "timed out while validating watch folder path")
			return
		}
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

	sem := acquirePathValidationSlot()
	select {
	case sem <- struct{}{}:
		// acquired slot
	case <-validateCtx.Done():
		return context.DeadlineExceeded
	}

	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		defer func() { <-sem }()
		info, err := statFn(path)
		if err != nil {
			if os.IsNotExist(err) {
				ch <- result{errors.New("folder not found: " + path)}
				return
			}
			// Don't expose raw OS error details to client; log them server-side.
			slog.WarnContext(ctx, "watch folder path validation failed",
				slog.String(otelkeys.WatchFolderPath, path),
				slog.Any(otelkeys.Error, err),
			)
			ch <- result{errors.New("unable to access path: " + path)}
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
		return context.DeadlineExceeded
	case r := <-ch:
		return r.err
	}
}

// getSettingOrEmpty returns the setting value for key, or "" if not found.
func (h *ConfigHandler) getSettingOrEmpty(ctx context.Context, key string) string {
	val, err := h.DB.GetSetting(ctx, key)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "failed to read setting",
				slog.String(otelkeys.Key, key),
				slog.Any(otelkeys.Error, err),
			)
		}
		return ""
	}
	return val
}
