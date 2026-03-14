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
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
)

// LibraryHandler holds dependencies for library endpoints.
type LibraryHandler struct {
	DB       *db.DB
	Enqueuer jobs.Enqueuer
	wg       sync.WaitGroup
}

// Wait blocks until all outstanding background enqueue goroutines complete.
func (h *LibraryHandler) Wait() { h.wg.Wait() }

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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleLibrary handles GET/PUT/DELETE /api/libraries/{id}.
func (h *LibraryHandler) HandleLibrary(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/libraries/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid library ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getLibrary(w, r, id)
	case http.MethodPut:
		h.updateLibrary(w, r, id)
	case http.MethodDelete:
		h.deleteLibrary(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *LibraryHandler) listLibraries(w http.ResponseWriter, r *http.Request) {
	slog.DebugContext(r.Context(), "listing libraries")
	libraries, err := h.DB.ListLibraries()
	if err != nil {
		slog.Error("failed to list libraries", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list libraries")
		return
	}

	slog.DebugContext(r.Context(), "libraries listed", slog.Int("count", len(libraries)))

	dtos := make([]libraryDTO, 0, len(libraries))
	for i := range libraries {
		dtos = append(dtos, toLibraryDTO(&libraries[i]))
	}

	writeJSON(w, http.StatusOK, dtos)
}

// validateAndPrepareLibrary validates the library request fields and encodes paths to JSON.
// It writes the appropriate error response and returns ("", false) on failure.
func validateAndPrepareLibrary(w http.ResponseWriter, req *libraryRequest) (pathsJSON string, ok bool) {
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return "", false
	}
	if len(req.Paths) == 0 {
		writeError(w, http.StatusBadRequest, "at least one path is required")
		return "", false
	}
	if slices.Contains(req.Paths, "") {
		writeError(w, http.StatusBadRequest, "paths must not be empty strings")
		return "", false
	}
	if err := validatePaths(req.Paths); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return "", false
	}
	if req.OrganizationType == "" {
		req.OrganizationType = "book_per_folder"
	}
	data, err := json.Marshal(req.Paths)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode paths")
		return "", false
	}
	return string(data), true
}

func (h *LibraryHandler) createLibrary(w http.ResponseWriter, r *http.Request) {
	var req libraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pathsJSON, ok := validateAndPrepareLibrary(w, &req)
	if !ok {
		return
	}

	slog.DebugContext(r.Context(), "creating library", slog.String("name", req.Name))

	lib, err := h.DB.CreateLibrary(req.Name, pathsJSON, req.OrganizationType, req.Monitored)
	if err != nil {
		if err == db.ErrLibraryNameExists {
			writeError(w, http.StatusConflict, "a library with that name already exists")
			return
		}
		slog.Error("failed to create library", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create library")
		return
	}

	dto := toLibraryDTO(lib)

	if h.Enqueuer != nil && len(dto.Paths) > 0 {
		paths := dto.Paths
		libID := lib.ID
		enqueuer := h.Enqueuer
		h.wg.Add(1)
		go func() {
			defer h.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			jobs.ScanLibrary(ctx, enqueuer, libID, paths)
		}()
	}

	writeJSON(w, http.StatusCreated, dto)
}

func (h *LibraryHandler) getLibrary(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching library", slog.String("library_id", id))
	lib, err := h.DB.GetLibrary(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "library not found")
			return
		}
		slog.Error("failed to get library", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get library")
		return
	}

	writeJSON(w, http.StatusOK, toLibraryDTO(lib))
}

func (h *LibraryHandler) updateLibrary(w http.ResponseWriter, r *http.Request, id string) {
	var req libraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pathsJSON, ok := validateAndPrepareLibrary(w, &req)
	if !ok {
		return
	}

	slog.DebugContext(r.Context(), "updating library", slog.String("library_id", id), slog.String("name", req.Name))

	lib, err := h.DB.UpdateLibrary(id, req.Name, pathsJSON, req.OrganizationType, req.Monitored)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "library not found")
			return
		}
		if err == db.ErrLibraryNameExists {
			writeError(w, http.StatusConflict, "a library with that name already exists")
			return
		}
		slog.Error("failed to update library", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update library")
		return
	}

	writeJSON(w, http.StatusOK, toLibraryDTO(lib))
}

func (h *LibraryHandler) deleteLibrary(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting library", slog.String("library_id", id))
	err := h.DB.DeleteLibrary(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "library not found")
			return
		}
		slog.Error("failed to delete library", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete library")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
