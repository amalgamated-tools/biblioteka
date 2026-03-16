package handlers

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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

type bookListDTO struct {
	Books  []bookSummaryDTO `json:"books"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
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

func (h *BookHandler) toBookDTO(ctx context.Context, b *db.Book) (bookDTO, error) {
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

	authors, err := h.DB.GetBookAuthors(ctx, b.ID)
	if err != nil {
		return bookDTO{}, err
	}
	for i := range authors {
		dto.Authors = append(dto.Authors, toAuthorDTO(&authors[i]))
	}

	entries, err := h.DB.GetBookSeries(ctx, b.ID)
	if err != nil {
		return bookDTO{}, err
	}
	for _, e := range entries {
		dto.Series = append(dto.Series, bookSeriesEntryDTO{
			Series:   toSeriesDTO(&e.Series),
			Position: e.Position,
		})
	}

	files, err := h.DB.ListBookFiles(ctx, b.ID)
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
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleBookRoutes dispatches /api/books/{id} and /api/books/{id}/{sub-resource}.
func (h *BookHandler) HandleBookRoutes(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/books/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid book ID")
		return
	}

	switch sub {
	case "":
		h.handleBook(w, r, id)
	case "authors":
		switch r.Method {
		case http.MethodGet:
			h.getBookAuthors(w, r, id)
		case http.MethodPut:
			h.putBookAuthors(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "series":
		switch r.Method {
		case http.MethodGet:
			h.getBookSeries(w, r, id)
		case http.MethodPut:
			h.putBookSeries(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "files":
		switch r.Method {
		case http.MethodGet:
			h.getBookFiles(w, r, id)
		case http.MethodPost:
			h.postBookFiles(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
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
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listBooks godoc
//
//	@Summary		List books
//	@Description	Returns paginated books (summary without relations)
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int	false	"Number of items to skip (default 0)"
//	@Success		200		{object}	bookListDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books [get]
func (h *BookHandler) listBooks(w http.ResponseWriter, r *http.Request) {
	const defaultLimit = 50
	const maxLimit = 200

	limit := defaultLimit
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid limit parameter")
			return
		}
		if n > maxLimit {
			n = maxLimit
		}
		limit = n
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid offset parameter")
			return
		}
		offset = n
	}

	slog.DebugContext(r.Context(), "listing books",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	books, total, err := h.DB.ListBooksPaginated(r.Context(), limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list books", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list books")
		return
	}

	slog.DebugContext(r.Context(), "books listed", slog.Int(otelkeys.Count, len(books)))

	dtos := make([]bookSummaryDTO, 0, len(books))
	for i := range books {
		dtos = append(dtos, toBookSummaryDTO(&books[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, bookListDTO{
		Books:  dtos,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// createBook godoc
//
//	@Summary		Create a book
//	@Description	Create a new book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		bookRequest	true	"Book data"
//	@Success		201		{object}	bookDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books [post]
func (h *BookHandler) createBook(w http.ResponseWriter, r *http.Request) {
	var req bookRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if req.Title == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "title is required")
		return
	}

	slog.DebugContext(r.Context(), "creating book", slog.String(otelkeys.Title, req.Title))

	b, err := h.DB.CreateBook(r.Context(), req.Title, req.Description, req.ASIN, req.ISBN10, req.ISBN13, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.PublicationDate, req.Publisher, req.Language, req.NumPages, req.CoverImageURL)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create book", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create book")
		return
	}

	dto, err := h.toBookDTO(r.Context(), b)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to build book DTO",
			slog.String(otelkeys.BookID, b.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create book")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionBookCreated, "book", b.ID, map[string]any{"title": b.Title}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	writeJSON(r.Context(), w, http.StatusCreated, dto)
}

// getBook godoc
//
//	@Summary		Get a book
//	@Description	Returns a single book with authors, series, and files
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{object}	bookDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id} [get]
func (h *BookHandler) getBook(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching book", slog.String(otelkeys.BookID, id))
	b, err := h.DB.GetBook(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get book", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get book")
		return
	}

	dto, err := h.toBookDTO(r.Context(), b)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to build book DTO",
			slog.String(otelkeys.BookID, b.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get book")
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, dto)
}

// updateBook godoc
//
//	@Summary		Update a book
//	@Description	Update an existing book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string		true	"Book ID"
//	@Param			body	body		bookRequest	true	"Book data"
//	@Success		200		{object}	bookDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books/{id} [put]
func (h *BookHandler) updateBook(w http.ResponseWriter, r *http.Request, id string) {
	var req bookRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if req.Title == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "title is required")
		return
	}

	slog.DebugContext(r.Context(), "updating book",
		slog.String(otelkeys.BookID, id),
		slog.String(otelkeys.Title, req.Title),
	)

	b, err := h.DB.UpdateBook(r.Context(), id, req.Title, req.Description, req.ASIN, req.ISBN10, req.ISBN13, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID, req.PublicationDate, req.Publisher, req.Language, req.NumPages, req.CoverImageURL)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to update book", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update book")
		return
	}

	dto, err := h.toBookDTO(r.Context(), b)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to build book DTO",
			slog.String(otelkeys.BookID, b.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update book")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionBookUpdated, "book", b.ID, map[string]any{"title": b.Title}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	writeJSON(r.Context(), w, http.StatusOK, dto)
}

// deleteBook godoc
//
//	@Summary		Delete a book
//	@Description	Delete a book by ID
//	@Tags			Books
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id} [delete]
func (h *BookHandler) deleteBook(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting book", slog.String(otelkeys.BookID, id))

	book, err := h.DB.GetBook(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get book", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	if err := h.DB.DeleteBook(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "book not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete book", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionBookDeleted, "book", id, map[string]any{"title": book.Title}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.WriteHeader(http.StatusNoContent)
}
