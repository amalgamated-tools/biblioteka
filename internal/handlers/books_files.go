package handlers

import (
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// getBookFiles godoc
//
//	@Summary		List book files
//	@Description	List files for a book
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{array}		bookFileDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/files [get]
func (h *BookHandler) getBookFiles(w http.ResponseWriter, r *http.Request, bookID string) {
	files, err := h.DB.ListBookFiles(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list book files", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list book files")
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, mapSlice(files, toBookFileDTO))
}

// createBookFileRequest is the request body for creating a book file.
type createBookFileRequest struct {
	FileType string  `json:"file_type"`
	FileName string  `json:"file_name"`
	FileSize int64   `json:"file_size"`
	FileHash *string `json:"file_hash"`
	FilePath string  `json:"file_path"`
}

// postBookFiles godoc
//
//	@Summary		Add a book file
//	@Description	Add a new file for a book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string					true	"Book ID"
//	@Param			body	body		createBookFileRequest	true	"Book file data"
//	@Success		201		{object}	bookFileDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books/{id}/files [post]
func (h *BookHandler) postBookFiles(w http.ResponseWriter, r *http.Request, bookID string) {
	var req createBookFileRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if req.FileType == "" || req.FileName == "" || req.FilePath == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "file_type, file_name, and file_path are required")
		return
	}
	bf, err := h.DB.CreateBookFile(r.Context(), bookID, req.FileType, req.FileName, req.FileSize, req.FileHash, req.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create book file", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create book file")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionBookFileCreated, "book_file", bf.ID, map[string]any{"book_id": bookID, "file_name": bf.FileName, "file_type": bf.FileType})

	writeJSON(r.Context(), w, http.StatusCreated, toBookFileDTO(bf))
}
