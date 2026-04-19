package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	// maxUploadSize caps the total multipart body including file content.
	maxUploadSize = 500 << 20 // 500 MB

	// maxUploadMemory limits the in-memory portion of multipart parsing;
	// anything beyond this is spilled to temporary files.
	maxUploadMemory = 32 << 20 // 32 MB

	// uploadStagingDir is the subdirectory within the library root used for
	// temporarily staging uploaded files until the background job processes them.
	uploadStagingDir = ".uploads"
)

// uploadAcceptedResponse is the JSON body returned on a successful upload.
type uploadAcceptedResponse struct {
	Message   string `json:"message"`
	FileName  string `json:"file_name"`
	FileType  string `json:"file_type"`
	LibraryID string `json:"library_id"`
}

// HandleUpload handles POST /api/books/upload.
// It accepts a multipart/form-data request containing a book file and optional
// metadata override fields, stages the file inside the target library's root
// directory, and enqueues a background process:file job that performs metadata
// extraction, file organisation, and database record creation.
//
//	@Summary		Upload a book file
//	@Description	Upload a book file (.epub, .mobi, .azw3, .pdf) to a library. The file is staged and processed asynchronously.
//	@Tags			Books
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file		formData	file	true	"Book file (.epub, .mobi, .azw3, .pdf)"
//	@Param			library_id	formData	string	true	"Target library ID"
//	@Param			title		formData	string	false	"Title override (takes precedence over extracted metadata)"
//	@Param			author		formData	string	false	"Author override (takes precedence over extracted metadata)"
//	@Param			description	formData	string	false	"Description override"
//	@Param			isbn		formData	string	false	"ISBN override (ISBN-10 or ISBN-13)"
//	@Param			language	formData	string	false	"Language override"
//	@Param			publisher	formData	string	false	"Publisher override"
//	@Success		202			{object}	uploadAcceptedResponse
//	@Failure		400			{object}	errorResponse
//	@Failure		401			{object}	errorResponse
//	@Failure		404			{object}	errorResponse
//	@Failure		413			{object}	errorResponse
//	@Failure		503			{object}	errorResponse
//	@Router			/books/upload [post]
func (h *BookHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.Enqueuer == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "background processing not configured")
		return
	}

	// Cap the entire request body before parsing to prevent resource exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadMemory); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(r.Context(), w, http.StatusRequestEntityTooLarge, "file too large")
			return
		}
		writeError(r.Context(), w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	libraryID := strings.TrimSpace(r.FormValue("library_id"))
	if libraryID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "library_id is required")
		return
	}

	lib, err := h.DB.GetLibrary(r.Context(), libraryID)
	if handleDBErr(r.Context(), w, err, "library") {
		return
	}

	libraryRoot, err := parseFirstLibraryPath(lib.Paths)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to parse library paths",
			slog.String(otelkeys.LibraryID, libraryID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "invalid library configuration")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Clean user-controlled filename so it cannot contain path separators.
	filename := filepath.Base(header.Filename)
	fileType, ok := detectUploadFileType(filename)
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "unsupported file type: must be one of "+strings.Join(jobs.SupportedFileTypes(), ", "))
		return
	}

	// Validate ISBN synchronously so the client gets immediate feedback
	// before the file is staged to disk.
	isbnRaw := strings.TrimSpace(r.FormValue("isbn"))
	if isbnRaw != "" {
		if exif.NormalizeISBN(isbnRaw) == "" {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid isbn: must be a valid ISBN-10 or ISBN-13")
			return
		}
	}

	stagingDir := filepath.Join(libraryRoot, uploadStagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		slog.ErrorContext(r.Context(), "failed to create upload staging directory",
			slog.String(otelkeys.Path, stagingDir),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to prepare upload staging area")
		return
	}

	prefix, err := generateRandomHex(8) // 16-char hex prefix avoids filename collisions
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to generate staging filename prefix",
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to prepare upload")
		return
	}

	stagingPath := filepath.Join(stagingDir, prefix+"_"+filename)

	fileSize, err := saveUploadedFile(file, stagingPath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to save uploaded file",
			slog.String(otelkeys.Path, stagingPath),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to save uploaded file")
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	payload := jobs.ProcessFilePayload{
		Path:                stagingPath,
		FileName:            filename,
		FileType:            fileType,
		FileSize:            fileSize,
		LibraryID:           libraryID,
		LibraryRoot:         libraryRoot,
		UserID:              userID,
		OverrideTitle:       strings.TrimSpace(r.FormValue("title")),
		OverrideAuthor:      strings.TrimSpace(r.FormValue("author")),
		OverrideDescription: strings.TrimSpace(r.FormValue("description")),
		OverrideISBN:        strings.TrimSpace(r.FormValue("isbn")),
		OverrideLanguage:    strings.TrimSpace(r.FormValue("language")),
		OverridePublisher:   strings.TrimSpace(r.FormValue("publisher")),
	}

	enqueueCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
	defer cancel()

	if _, err := h.Enqueuer.Enqueue(enqueueCtx, jobs.JobProcessFile, payload); err != nil {
		slog.ErrorContext(r.Context(), "failed to enqueue process:file job",
			slog.String(otelkeys.FileName, filename),
			slog.String(otelkeys.LibraryID, libraryID),
			slog.Any(otelkeys.Error, err),
		)
		// Best-effort cleanup of the staged file to avoid accumulating orphans.
		if rmErr := os.Remove(stagingPath); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.WarnContext(r.Context(), "failed to remove staged file after enqueue failure",
				slog.String(otelkeys.Path, stagingPath),
				slog.Any(otelkeys.Error, rmErr),
			)
		}
		writeError(r.Context(), w, http.StatusServiceUnavailable, "failed to queue file for processing")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionBookUploaded, "book_upload", stagingPath, map[string]any{
		"file_name":  filename,
		"file_type":  fileType,
		"file_size":  fileSize,
		"library_id": libraryID,
	})

	slog.InfoContext(r.Context(), "book file upload accepted",
		slog.String(otelkeys.FileName, filename),
		slog.String(otelkeys.FileType, fileType),
		slog.String(otelkeys.LibraryID, libraryID),
		slog.Int64(otelkeys.FileSize, fileSize),
	)

	writeJSON(r.Context(), w, http.StatusAccepted, uploadAcceptedResponse{
		Message:   "file accepted for processing",
		FileName:  filename,
		FileType:  fileType,
		LibraryID: libraryID,
	})
}

// detectUploadFileType returns the normalised file-type label for filename based
// on its extension. It returns the empty string and false for unsupported types.
func detectUploadFileType(filename string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	ft, ok := jobs.SupportedExtensions[ext]
	return ft, ok
}

// parseFirstLibraryPath decodes a JSON-encoded library paths array and returns
// the first path. Returns an error when the JSON is invalid or the list is empty.
func parseFirstLibraryPath(paths string) (string, error) {
	var list []string
	if err := json.Unmarshal([]byte(paths), &list); err != nil {
		return "", fmt.Errorf("parse library paths JSON: %w", err)
	}
	if len(list) == 0 {
		return "", errors.New("library has no paths configured")
	}
	return list[0], nil
}

// saveUploadedFile copies src to the file at destPath, creating it if it does not
// exist. It returns the number of bytes written. On any error the partially
// written destination file is removed.
func saveUploadedFile(src io.Reader, destPath string) (int64, error) {
	dst, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create staging file: %w", err)
	}

	n, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()

	if copyErr != nil {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("write staged file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("close staged file: %w", closeErr)
	}

	return n, nil
}
