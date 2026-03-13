package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// BookHandler holds dependencies for book endpoints.
type BookHandler struct {
	DB *db.DB
}

type bookRequest struct {
	Title           string  `json:"title"`
	Description     *string `json:"description"`
	ASIN            *string `json:"asin"`
	ISBN10          *string `json:"isbn10"`
	ISBN13          *string `json:"isbn13"`
	GoodreadsID     *string `json:"goodreads_id"`
	HardcoverID     *string `json:"hardcover_id"`
	GoogleBooksID   *string `json:"google_books_id"`
	PublicationDate *string `json:"publication_date"`
	Publisher       *string `json:"publisher"`
	Language        *string `json:"language"`
	NumPages        *int    `json:"num_pages"`
	CoverImageURL   *string `json:"cover_image_url"`
}

type bookSeriesEntryDTO struct {
	Series   seriesDTO `json:"series"`
	Position *float64  `json:"position"`
}

type bookFileDTO struct {
	ID        string       `json:"id"`
	BookID    string       `json:"book_id"`
	FileType  string       `json:"file_type"`
	FileName  string       `json:"file_name"`
	FileSize  int64        `json:"file_size"`
	FileHash  *string      `json:"file_hash"`
	FilePath  string       `json:"file_path"`
	CreatedAt db.Timestamp `json:"created_at"`
	UpdatedAt db.Timestamp `json:"updated_at"`
}

func toBookFileDTO(bf *db.BookFile) bookFileDTO {
	return bookFileDTO{
		ID:        bf.ID,
		BookID:    bf.BookID,
		FileType:  bf.FileType,
		FileName:  bf.FileName,
		FileSize:  bf.FileSize,
		FileHash:  bf.FileHash,
		FilePath:  bf.FilePath,
		CreatedAt: bf.CreatedAt,
		UpdatedAt: bf.UpdatedAt,
	}
}

// bookSummaryDTO contains core book fields without related entities.
// Used in list endpoints to avoid N+1 queries.
type bookSummaryDTO struct {
	ID              string       `json:"id"`
	Title           string       `json:"title"`
	Description     *string      `json:"description"`
	ASIN            *string      `json:"asin"`
	ISBN10          *string      `json:"isbn10"`
	ISBN13          *string      `json:"isbn13"`
	GoodreadsID     *string      `json:"goodreads_id"`
	HardcoverID     *string      `json:"hardcover_id"`
	GoogleBooksID   *string      `json:"google_books_id"`
	PublicationDate *string      `json:"publication_date"`
	Publisher       *string      `json:"publisher"`
	Language        *string      `json:"language"`
	NumPages        *int         `json:"num_pages"`
	CoverImageURL   *string      `json:"cover_image_url"`
	CreatedAt       db.Timestamp `json:"created_at"`
	UpdatedAt       db.Timestamp `json:"updated_at"`
}

func toBookSummaryDTO(b *db.Book) bookSummaryDTO {
	return bookSummaryDTO{
		ID:              b.ID,
		Title:           b.Title,
		Description:     b.Description,
		ASIN:            b.ASIN,
		ISBN10:          b.ISBN10,
		ISBN13:          b.ISBN13,
		GoodreadsID:     b.GoodreadsID,
		HardcoverID:     b.HardcoverID,
		GoogleBooksID:   b.GoogleBooksID,
		PublicationDate: b.PublicationDate,
		Publisher:       b.Publisher,
		Language:        b.Language,
		NumPages:        b.NumPages,
		CoverImageURL:   b.CoverImageURL,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
}

// bookDTO contains full book details including related entities.
// Used in single-book endpoints (GET/POST/PUT by ID).
type bookDTO struct {
	ID              string               `json:"id"`
	Title           string               `json:"title"`
	Description     *string              `json:"description"`
	ASIN            *string              `json:"asin"`
	ISBN10          *string              `json:"isbn10"`
	ISBN13          *string              `json:"isbn13"`
	GoodreadsID     *string              `json:"goodreads_id"`
	HardcoverID     *string              `json:"hardcover_id"`
	GoogleBooksID   *string              `json:"google_books_id"`
	PublicationDate *string              `json:"publication_date"`
	Publisher       *string              `json:"publisher"`
	Language        *string              `json:"language"`
	NumPages        *int                 `json:"num_pages"`
	CoverImageURL   *string              `json:"cover_image_url"`
	Authors         []authorDTO          `json:"authors"`
	Series          []bookSeriesEntryDTO `json:"series"`
	Files           []bookFileDTO        `json:"files"`
	CreatedAt       db.Timestamp         `json:"created_at"`
	UpdatedAt       db.Timestamp         `json:"updated_at"`
}

func (h *BookHandler) toBookDTO(b *db.Book) (bookDTO, error) {
	dto := bookDTO{
		ID:              b.ID,
		Title:           b.Title,
		Description:     b.Description,
		ASIN:            b.ASIN,
		ISBN10:          b.ISBN10,
		ISBN13:          b.ISBN13,
		GoodreadsID:     b.GoodreadsID,
		HardcoverID:     b.HardcoverID,
		GoogleBooksID:   b.GoogleBooksID,
		PublicationDate: b.PublicationDate,
		Publisher:       b.Publisher,
		Language:        b.Language,
		NumPages:        b.NumPages,
		CoverImageURL:   b.CoverImageURL,
		Authors:         []authorDTO{},
		Series:          []bookSeriesEntryDTO{},
		Files:           []bookFileDTO{},
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}

	authors, err := h.DB.GetBookAuthors(b.ID)
	if err != nil {
		return bookDTO{}, err
	}
	for i := range authors {
		dto.Authors = append(dto.Authors, toAuthorDTO(&authors[i]))
	}

	entries, err := h.DB.GetBookSeries(b.ID)
	if err != nil {
		return bookDTO{}, err
	}
	for _, e := range entries {
		dto.Series = append(dto.Series, bookSeriesEntryDTO{
			Series:   toSeriesDTO(&e.Series),
			Position: e.Position,
		})
	}

	files, err := h.DB.ListBookFiles(b.ID)
	if err != nil {
		return bookDTO{}, err
	}
	for i := range files {
		dto.Files = append(dto.Files, toBookFileDTO(&files[i]))
	}

	return dto, nil
}

// HandleBooks handles GET /api/books and POST /api/books.
func (h *BookHandler) HandleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBooks(w, r)
	case http.MethodPost:
		h.createBook(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleBookRoutes dispatches /api/books/{id} and /api/books/{id}/{sub-resource}.
func (h *BookHandler) HandleBookRoutes(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/books/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid book ID")
		return
	}

	switch sub {
	case "":
		h.handleBook(w, r, id)
	case "authors":
		h.handleBookAuthors(w, r, id)
	case "series":
		h.handleBookSeries(w, r, id)
	case "files":
		h.handleBookFiles(w, r, id)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *BookHandler) handleBook(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		h.getBook(w, r, id)
	case http.MethodPut:
		h.updateBook(w, r, id)
	case http.MethodDelete:
		h.deleteBook(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *BookHandler) listBooks(w http.ResponseWriter, _ *http.Request) {
	books, err := h.DB.ListBooks()
	if err != nil {
		slog.Error("failed to list books", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list books")
		return
	}

	dtos := make([]bookSummaryDTO, 0, len(books))
	for i := range books {
		dtos = append(dtos, toBookSummaryDTO(&books[i]))
	}

	writeJSON(w, http.StatusOK, dtos)
}

func (h *BookHandler) createBook(w http.ResponseWriter, r *http.Request) {
	var req bookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	b, err := h.DB.CreateBook(req.Title, req.Description, req.ASIN, req.ISBN10, req.ISBN13, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.PublicationDate, req.Publisher, req.Language, req.NumPages, req.CoverImageURL)
	if err != nil {
		slog.Error("failed to create book", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create book")
		return
	}

	dto, err := h.toBookDTO(b)
	if err != nil {
		slog.Error("failed to build book DTO", "book_id", b.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create book")
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *BookHandler) getBook(w http.ResponseWriter, _ *http.Request, id string) {
	b, err := h.DB.GetBook(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "book not found")
			return
		}
		slog.Error("failed to get book", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get book")
		return
	}

	dto, err := h.toBookDTO(b)
	if err != nil {
		slog.Error("failed to build book DTO", "book_id", b.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get book")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *BookHandler) updateBook(w http.ResponseWriter, r *http.Request, id string) {
	var req bookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	b, err := h.DB.UpdateBook(id, req.Title, req.Description, req.ASIN, req.ISBN10, req.ISBN13, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.PublicationDate, req.Publisher, req.Language, req.NumPages, req.CoverImageURL)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "book not found")
			return
		}
		slog.Error("failed to update book", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	dto, err := h.toBookDTO(b)
	if err != nil {
		slog.Error("failed to build book DTO", "book_id", b.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update book")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
func (h *BookHandler) deleteBook(w http.ResponseWriter, _ *http.Request, id string) {
	err := h.DB.DeleteBook(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "book not found")
			return
		}
		slog.Error("failed to delete book", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleBookAuthors handles GET/PUT /api/books/{id}/authors.
func (h *BookHandler) handleBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
	switch r.Method {
	case http.MethodGet:
		authors, err := h.DB.GetBookAuthors(bookID)
		if err != nil {
			slog.Error("failed to get book authors", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to get book authors")
			return
		}
		dtos := make([]authorDTO, 0, len(authors))
		for i := range authors {
			dtos = append(dtos, toAuthorDTO(&authors[i]))
		}
		writeJSON(w, http.StatusOK, dtos)

	case http.MethodPut:
		var req struct {
			AuthorIDs []string `json:"author_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.DB.SetBookAuthors(bookID, req.AuthorIDs); err != nil {
			slog.Error("failed to set book authors", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to set book authors")
			return
		}
		authors, err := h.DB.GetBookAuthors(bookID)
		if err != nil {
			slog.Error("failed to get book authors", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to get book authors")
			return
		}
		dtos := make([]authorDTO, 0, len(authors))
		for i := range authors {
			dtos = append(dtos, toAuthorDTO(&authors[i]))
		}
		writeJSON(w, http.StatusOK, dtos)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleBookSeries handles GET/PUT /api/books/{id}/series.
func (h *BookHandler) handleBookSeries(w http.ResponseWriter, r *http.Request, bookID string) {
	switch r.Method {
	case http.MethodGet:
		entries, err := h.DB.GetBookSeries(bookID)
		if err != nil {
			slog.Error("failed to get book series", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to get book series")
			return
		}
		dtos := make([]bookSeriesEntryDTO, 0, len(entries))
		for _, e := range entries {
			dtos = append(dtos, bookSeriesEntryDTO{
				Series:   toSeriesDTO(&e.Series),
				Position: e.Position,
			})
		}
		writeJSON(w, http.StatusOK, dtos)

	case http.MethodPut:
		var req struct {
			Entries []db.BookSeriesInput `json:"entries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.DB.SetBookSeries(bookID, req.Entries); err != nil {
			slog.Error("failed to set book series", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to set book series")
			return
		}
		entries, err := h.DB.GetBookSeries(bookID)
		if err != nil {
			slog.Error("failed to get book series", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to get book series")
			return
		}
		dtos := make([]bookSeriesEntryDTO, 0, len(entries))
		for _, e := range entries {
			dtos = append(dtos, bookSeriesEntryDTO{
				Series:   toSeriesDTO(&e.Series),
				Position: e.Position,
			})
		}
		writeJSON(w, http.StatusOK, dtos)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleBookFiles handles GET/POST /api/books/{id}/files.
func (h *BookHandler) handleBookFiles(w http.ResponseWriter, r *http.Request, bookID string) {
	switch r.Method {
	case http.MethodGet:
		files, err := h.DB.ListBookFiles(bookID)
		if err != nil {
			slog.Error("failed to list book files", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to list book files")
			return
		}
		dtos := make([]bookFileDTO, 0, len(files))
		for i := range files {
			dtos = append(dtos, toBookFileDTO(&files[i]))
		}
		writeJSON(w, http.StatusOK, dtos)

	case http.MethodPost:
		var req struct {
			FileType string  `json:"file_type"`
			FileName string  `json:"file_name"`
			FileSize int64   `json:"file_size"`
			FileHash *string `json:"file_hash"`
			FilePath string  `json:"file_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.FileType == "" || req.FileName == "" || req.FilePath == "" {
			writeError(w, http.StatusBadRequest, "file_type, file_name, and file_path are required")
			return
		}
		bf, err := h.DB.CreateBookFile(bookID, req.FileType, req.FileName, req.FileSize, req.FileHash, req.FilePath)
		if err != nil {
			slog.Error("failed to create book file", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create book file")
			return
		}
		writeJSON(w, http.StatusCreated, toBookFileDTO(bf))

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
