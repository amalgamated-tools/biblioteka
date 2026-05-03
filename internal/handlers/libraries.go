package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// LibraryHandler holds dependencies for library endpoints.
type LibraryHandler struct {
	DB       *db.DB
	Enqueuer jobs.Enqueuer
}

type libraryRequest struct {
	Name             string   `json:"name"`
	Paths            []string `json:"paths"`
	OrganizationType string   `json:"organization_type"`
	Monitored        bool     `json:"monitored"`
}

type libraryDTO struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Paths            []string     `json:"paths"`
	OrganizationType string       `json:"organization_type"`
	Monitored        bool         `json:"monitored"`
	CreatedAt        db.Timestamp `json:"created_at"`
	UpdatedAt        db.Timestamp `json:"updated_at"`
}

func toLibraryDTO(lib *db.Library) libraryDTO {
	var paths []string
	if err := json.Unmarshal([]byte(lib.Paths), &paths); err != nil {
		paths = []string{}
	}
	return libraryDTO{
		ID:               lib.ID,
		Name:             lib.Name,
		Paths:            paths,
		OrganizationType: lib.OrganizationType,
		Monitored:        lib.Monitored,
		CreatedAt:        lib.CreatedAt,
		UpdatedAt:        lib.UpdatedAt,
	}
}

// HandleLibraries handles GET /api/libraries and POST /api/libraries.
func (h *LibraryHandler) HandleLibraries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listLibraries(w, r)
	case http.MethodPost:
		h.createLibrary(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleLibrary handles GET/PUT/DELETE /api/libraries/{id} and sub-resources like /api/libraries/{id}/books.
func (h *LibraryHandler) HandleLibrary(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/libraries/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid library ID")
		return
	}

	switch sub {
	case "":
		switch r.Method {
		case http.MethodGet:
			h.getLibrary(w, r, id)
		case http.MethodPut:
			h.updateLibrary(w, r, id)
		case http.MethodDelete:
			h.deleteLibrary(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "books":
		switch r.Method {
		case http.MethodGet:
			h.listLibraryBooks(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

// listLibraries returns all libraries.
//
//	@Summary		List libraries
//	@Description	Returns all libraries
//	@Tags			Libraries
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		libraryDTO
//	@Failure		500	{object}	errorResponse
//	@Router			/libraries [get]
func (h *LibraryHandler) listLibraries(w http.ResponseWriter, r *http.Request) {
	listEntities(w, r, "libraries", h.DB.ListLibraries, toLibraryDTO)
}

func validateAndPrepareLibrary(ctx context.Context, w http.ResponseWriter, req *libraryRequest, defaultOrganizationType string) (pathsJSON string, ok bool) {
	if strings.TrimSpace(req.Name) == "" {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return "", false
	}
	if len(req.Paths) == 0 {
		writeError(ctx, w, http.StatusBadRequest, "at least one path is required")
		return "", false
	}
	if slices.Contains(req.Paths, "") {
		writeError(ctx, w, http.StatusBadRequest, "paths must not be empty strings")
		return "", false
	}
	validateCtx, cancel := context.WithTimeout(ctx, pathValidationTimeout)
	defer cancel()
	if err := validatePaths(validateCtx, req.Paths); err != nil {
		var pve *pathValidationError
		switch {
		case errors.As(err, &pve):
			writeError(ctx, w, http.StatusBadRequest, pve.Error())
		case errors.Is(err, context.DeadlineExceeded):
			slog.WarnContext(ctx, "library path validation timed out",
				slog.Any(otelkeys.Error, err),
				slog.Any(otelkeys.LibraryPaths, req.Paths),
			)
			writeError(ctx, w, http.StatusInternalServerError, "path validation timed out")
		case errors.Is(err, context.Canceled):
			// Client disconnected; no point writing a response.
			return "", false
		default:
			slog.ErrorContext(ctx, "failed to validate library paths",
				slog.Any(otelkeys.Error, err),
				slog.Any(otelkeys.LibraryPaths, req.Paths),
			)
			writeError(ctx, w, http.StatusInternalServerError, "failed to validate paths")
		}
		return "", false
	}
	if req.OrganizationType == "" {
		req.OrganizationType = defaultOrganizationType
	}
	if req.OrganizationType != "" && !db.IsValidLibraryOrganizationType(req.OrganizationType) {
		writeError(ctx, w, http.StatusBadRequest, "organization_type must be one of: "+strings.Join(db.LibraryOrganizationTypeNames(), ", "))
		return "", false
	}
	data, err := json.Marshal(req.Paths)
	if err != nil {
		writeError(ctx, w, http.StatusInternalServerError, "failed to encode paths")
		return "", false
	}
	return string(data), true
}

// createLibrary creates a new library and enqueues a scan job for each configured path (admin only).
//
//	@Summary		Create a library
//	@Description	Create a new library and enqueue scan jobs
//	@Tags			Libraries
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Failure		403		{object}	errorResponse
//	@Param			body	body		libraryRequest	true	"Library data"
//	@Success		201		{object}	libraryDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/libraries [post]
func (h *LibraryHandler) createLibrary(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	var req libraryRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	pathsJSON, ok := validateAndPrepareLibrary(r.Context(), w, &req, db.LibraryOrganizationBookPerFolder)
	if !ok {
		return
	}

	slog.DebugContext(r.Context(), "creating library", slog.String(otelkeys.Name, req.Name))

	lib, err := h.DB.CreateLibrary(r.Context(), req.Name, pathsJSON, req.OrganizationType, req.Monitored)
	if err != nil {
		if handleNameErr(r.Context(), w, err, db.ErrInvalidLibraryName, db.ErrLibraryNameExists, "a library") {
			return
		}
		slog.ErrorContext(r.Context(), "failed to create library", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create library")
		return
	}

	dto := toLibraryDTO(lib)

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionLibraryCreated, "library", lib.ID, map[string]any{"name": lib.Name})

	if h.Enqueuer != nil && len(dto.Paths) > 0 {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
		defer cancel()
		if _, err := h.Enqueuer.Enqueue(ctx, jobs.JobScanLibrary, jobs.ScanLibraryPayload{
			LibraryID: lib.ID,
			Paths:     dto.Paths,
		}); err != nil {
			slog.ErrorContext(r.Context(), "failed to enqueue scan:library job",
				slog.String(otelkeys.LibraryID, lib.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	writeJSON(r.Context(), w, http.StatusCreated, dto)
}

// getLibrary returns a single library by ID.
//
//	@Summary		Get a library
//	@Description	Returns a single library by ID
//	@Tags			Libraries
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Library ID"
//	@Success		200	{object}	libraryDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/libraries/{id} [get]
func (h *LibraryHandler) getLibrary(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching library", slog.String(otelkeys.LibraryID, id))
	lib, err := h.DB.GetLibrary(r.Context(), id)
	if handleDBErr(r.Context(), w, err, "library") {
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, toLibraryDTO(lib))
}

// updateLibrary replaces the configuration for an existing library (admin only).
//
//	@Summary		Update a library
//	@Description	Update an existing library
//	@Tags			Libraries
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Failure		403		{object}	errorResponse
//	@Param			id		path		string			true	"Library ID"
//	@Param			body	body		libraryRequest	true	"Library data"
//	@Success		200		{object}	libraryDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/libraries/{id} [put]
func (h *LibraryHandler) updateLibrary(w http.ResponseWriter, r *http.Request, id string) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	var req libraryRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	existing, err := h.DB.GetLibrary(r.Context(), id)
	if handleDBErr(r.Context(), w, err, "library") {
		return
	}

	pathsJSON, ok := validateAndPrepareLibrary(r.Context(), w, &req, existing.OrganizationType)
	if !ok {
		return
	}

	slog.DebugContext(r.Context(), "updating library",
		slog.String(otelkeys.LibraryID, id),
		slog.String(otelkeys.Name, req.Name),
	)

	lib, err := h.DB.UpdateLibrary(r.Context(), id, req.Name, pathsJSON, req.OrganizationType, req.Monitored)
	if handleUpdateErr(r.Context(), w, err, db.ErrInvalidLibraryName, db.ErrLibraryNameExists, "a library", "library", id) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionLibraryUpdated, "library", lib.ID, map[string]any{"name": lib.Name})

	writeJSON(r.Context(), w, http.StatusOK, toLibraryDTO(lib))
}

// deleteLibrary permanently removes a library (admin only).
//
//	@Summary		Delete a library
//	@Description	Delete a library by ID
//	@Tags			Libraries
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Failure		403	{object}	errorResponse
//	@Param			id	path		string	true	"Library ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/libraries/{id} [delete]
func (h *LibraryHandler) deleteLibrary(w http.ResponseWriter, r *http.Request, id string) {
	if !requireAdmin(h.DB, w, r) {
		return
	}

	deleteResource(h.DB, w, r, id, "library", "library", otelkeys.LibraryID,
		h.DB.GetLibrary, h.DB.DeleteLibrary,
		db.AuditActionLibraryDeleted,
		func(l *db.Library) map[string]any { return map[string]any{"name": l.Name} },
	)
}

// listLibraryBooks returns paginated books belonging to the specified library.
//
//	@Summary		List books in a library
//	@Description	Returns paginated books belonging to a specific library
//	@Tags			Libraries
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Library ID"
//	@Param			limit	query		int		false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int		false	"Number of items to skip (default 0)"
//	@Success		200		{object}	bookListDTO
//	@Failure		401		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/libraries/{id}/books [get]
func (h *LibraryHandler) listLibraryBooks(w http.ResponseWriter, r *http.Request, id string) {
	listParentBooks(w, r, id,
		slog.String(otelkeys.LibraryID, id),
		h.DB.ListBooksByLibraryPaginated,
		h.DB.GetLibrary,
		"library",
	)
}

// pathValidationTimeout is the maximum time allowed to stat all library paths,
// guarding against blocking the HTTP handler goroutine on slow NFS/SMB mounts.
const pathValidationTimeout = 5 * time.Second

// validatePaths checks that every path in paths exists and is a directory,
// running the filesystem checks in a background goroutine so that
// pathValidationTimeout (via context) can abort a hung os.Stat call.
func validatePaths(ctx context.Context, paths []string) error {
	return validatePathsWith(ctx, paths, os.Stat)
}

// validatePathsWith is the testable core of validatePaths; callers may supply
// a custom stat function (e.g. a blocking stub in tests).
//
// Note: os.Stat is not context-cancelable. If statFn blocks indefinitely
// (e.g. an unresponsive NFS/SMB mount with no kernel-level timeout), the
// background goroutine will remain alive even after the caller returns via
// ctx.Done(). The context timeout bounds handler latency, not goroutine
// lifetime. If this becomes an operational concern, configure mount-level
// timeouts (e.g. NFS timeo=/retrans=/soft options) rather than trying to
// kill the goroutine from Go.
func validatePathsWith(ctx context.Context, paths []string, statFn func(string) (os.FileInfo, error)) error {
	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		for _, p := range paths {
			info, err := statFn(p)
			if err != nil {
				if os.IsNotExist(err) {
					ch <- result{&pathValidationError{msg: "folder not found: " + p}}
					return
				}
				ch <- result{err}
				return
			}
			if !info.IsDir() {
				ch <- result{&pathValidationError{msg: "path is not a folder: " + p}}
				return
			}
		}
		ch <- result{nil}
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("path validation timed out: %w", ctx.Err())
	case r := <-ch:
		return r.err
	}
}

// pathValidationError carries a user-safe message about an invalid library path.
type pathValidationError struct{ msg string }

func (e *pathValidationError) Error() string { return e.msg }
