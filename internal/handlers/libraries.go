package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
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

// listLibraries godoc
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
	slog.DebugContext(r.Context(), "listing libraries")
	libraries, err := h.DB.ListLibraries(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list libraries", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list libraries")
		return
	}

	slog.DebugContext(r.Context(), "libraries listed", slog.Int(otelkeys.Count, len(libraries)))

	dtos := make([]libraryDTO, 0, len(libraries))
	for i := range libraries {
		dtos = append(dtos, toLibraryDTO(&libraries[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

// validateAndPrepareLibrary validates the library request fields and encodes paths to JSON.
// It writes the appropriate error response and returns ("", false) on failure.
func validateAndPrepareLibrary(ctx context.Context, w http.ResponseWriter, req *libraryRequest) (pathsJSON string, ok bool) {
	if req.Name == "" {
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
	if err := validatePaths(req.Paths); err != nil {
		writeError(ctx, w, http.StatusBadRequest, err.Error())
		return "", false
	}
	if req.OrganizationType == "" {
		req.OrganizationType = "book_per_folder"
	}
	data, err := json.Marshal(req.Paths)
	if err != nil {
		writeError(ctx, w, http.StatusInternalServerError, "failed to encode paths")
		return "", false
	}
	return string(data), true
}

// createLibrary godoc
//
//	@Summary		Create a library
//	@Description	Create a new library and enqueue scan jobs
//	@Tags			Libraries
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		libraryRequest	true	"Library data"
//	@Success		201		{object}	libraryDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/libraries [post]
func (h *LibraryHandler) createLibrary(w http.ResponseWriter, r *http.Request) {
	var req libraryRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	pathsJSON, ok := validateAndPrepareLibrary(r.Context(), w, &req)
	if !ok {
		return
	}

	slog.DebugContext(r.Context(), "creating library", slog.String(otelkeys.Name, req.Name))

	lib, err := h.DB.CreateLibrary(r.Context(), req.Name, pathsJSON, req.OrganizationType, req.Monitored)
	if err != nil {
		if err == db.ErrLibraryNameExists {
			writeError(r.Context(), w, http.StatusConflict, "a library with that name already exists")
			return
		}
		slog.ErrorContext(r.Context(), "failed to create library", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create library")
		return
	}

	dto := toLibraryDTO(lib)

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionLibraryCreated, "library", lib.ID, map[string]any{"name": lib.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

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

// getLibrary godoc
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
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "library not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get library", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get library")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toLibraryDTO(lib))
}

// updateLibrary godoc
//
//	@Summary		Update a library
//	@Description	Update an existing library
//	@Tags			Libraries
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string			true	"Library ID"
//	@Param			body	body		libraryRequest	true	"Library data"
//	@Success		200		{object}	libraryDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/libraries/{id} [put]
func (h *LibraryHandler) updateLibrary(w http.ResponseWriter, r *http.Request, id string) {
	var req libraryRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	pathsJSON, ok := validateAndPrepareLibrary(r.Context(), w, &req)
	if !ok {
		return
	}

	slog.DebugContext(r.Context(), "updating library",
		slog.String(otelkeys.LibraryID, id),
		slog.String(otelkeys.Name, req.Name),
	)

	lib, err := h.DB.UpdateLibrary(r.Context(), id, req.Name, pathsJSON, req.OrganizationType, req.Monitored)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "library not found")
			return
		}
		if err == db.ErrLibraryNameExists {
			writeError(r.Context(), w, http.StatusConflict, "a library with that name already exists")
			return
		}
		slog.ErrorContext(r.Context(), "failed to update library",
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update library")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionLibraryUpdated, "library", lib.ID, map[string]any{"name": lib.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	writeJSON(r.Context(), w, http.StatusOK, toLibraryDTO(lib))
}

// deleteLibrary godoc
//
//	@Summary		Delete a library
//	@Description	Delete a library by ID
//	@Tags			Libraries
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Library ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/libraries/{id} [delete]
func (h *LibraryHandler) deleteLibrary(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting library", slog.String(otelkeys.LibraryID, id))

	lib, err := h.DB.GetLibrary(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "library not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get library", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete library")
		return
	}

	if err := h.DB.DeleteLibrary(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "library not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete library", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete library")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionLibraryDeleted, "library", id, map[string]any{"name": lib.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.WriteHeader(http.StatusNoContent)
}

// listLibraryBooks godoc
//
//	@Summary		List books in a library
//	@Description	Returns all books belonging to a specific library
//	@Tags			Libraries
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Library ID"
//	@Success		200	{array}		bookSummaryDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/libraries/{id}/books [get]
func (h *LibraryHandler) listLibraryBooks(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "listing library books", slog.String(otelkeys.LibraryID, id))

	books, err := h.DB.ListBooksByLibrary(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list library books",
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list library books")
		return
	}

	// If no books found, check whether the library actually exists.
	if len(books) == 0 {
		_, err := h.DB.GetLibrary(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				writeError(r.Context(), w, http.StatusNotFound, "library not found")
				return
			}
			slog.ErrorContext(r.Context(), "failed to get library",
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to get library")
			return
		}
	}

	slog.DebugContext(r.Context(), "library books listed", slog.Int(otelkeys.Count, len(books)))

	dtos := make([]bookSummaryDTO, 0, len(books))
	for i := range books {
		dtos = append(dtos, toBookSummaryDTO(&books[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

func validatePaths(paths []string) error {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("folder not found: %s", p)
		}
		if !info.IsDir() {
			return fmt.Errorf("path is not a folder: %s", p)
		}
	}
	return nil
}
