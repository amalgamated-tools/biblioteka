package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// listBooks returns a paginated list of books, optionally filtered by a search query.
//
//	@Summary		List books
//	@Description	Returns paginated books (summary without relations). When query is provided, performs a title/description search.
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Param			query	query		string	false	"Search query (SQLite: FTS5 full-text match across title and description; PostgreSQL: title/description substring match)"
//	@Param			limit	query		int		false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int		false	"Number of items to skip (default 0)"
//	@Success		200		{object}	bookListDTO
//	@Failure		401		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books [get]
func (h *BookHandler) listBooks(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)
	query := strings.TrimSpace(r.URL.Query().Get("query"))

	slog.DebugContext(r.Context(), "listing books",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
		slog.String(otelkeys.Query, query),
	)

	var (
		books []db.Book
		total int
		err   error
	)

	if query != "" {
		books, total, err = h.DB.SearchBooks(r.Context(), query, limit, offset)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to search books", slog.Any(otelkeys.Error, err))
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to search books")
			return
		}
	} else {
		books, total, err = h.DB.ListBooksPaginated(r.Context(), limit, offset)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to list books", slog.Any(otelkeys.Error, err))
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to list books")
			return
		}
	}

	slog.DebugContext(r.Context(), "books listed", slog.Int(otelkeys.Count, len(books)))

	writeJSON(r.Context(), w, http.StatusOK, bookListDTO{
		Books: mapSlice(books, toBookSummaryDTO),
		paginationMeta: paginationMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// createBook creates a new book record and enqueues a Goodreads enrichment job.
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

	b, err := h.DB.CreateBook(r.Context(), db.BookInput{
		Title:           req.Title,
		Description:     req.Description,
		ASIN:            req.ASIN,
		ISBN10:          req.ISBN10,
		ISBN13:          req.ISBN13,
		GoodreadsID:     req.GoodreadsID,
		HardcoverID:     req.HardcoverID,
		GoogleBooksID:   req.GoogleBooksID,
		PublicationDate: req.PublicationDate,
		Publisher:       req.Publisher,
		Language:        req.Language,
		CoverImageURL:   req.CoverImageURL,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create book", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create book")
		return
	}

	dto, err := h.loadBookDTO(r.Context(), b)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to build book DTO",
			slog.String(otelkeys.BookID, b.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create book")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionBookCreated, "book", b.ID, map[string]any{"title": b.Title})

	writeJSON(r.Context(), w, http.StatusCreated, dto)

	if h.Enqueuer != nil {
		enqueueCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
		defer cancel()
		if _, err := h.Enqueuer.Enqueue(enqueueCtx, jobs.JobEnrichGoodreads, jobs.EnrichGoodreadsPayload{
			BookID: b.ID,
			UserID: userID,
		}, jobs.WithUnique(24*time.Hour)); err != nil {
			slog.WarnContext(r.Context(), "failed to enqueue enrich:goodreads job",
				slog.String(otelkeys.BookID, b.ID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}
}

// getBook returns a single book with its authors, series, tags, and files.
//
//	@Summary		Get a book
//	@Description	Returns a single book with authors, series, tags, and files
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
	if handleDBErr(r.Context(), w, err, "book") {
		return
	}

	dto, err := h.loadBookDTO(r.Context(), b)
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

// updateBook replaces the metadata fields of an existing book.
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

	b, err := h.DB.UpdateBook(r.Context(), id, db.BookInput{
		Title:           req.Title,
		Description:     req.Description,
		ASIN:            req.ASIN,
		ISBN10:          req.ISBN10,
		ISBN13:          req.ISBN13,
		GoodreadsID:     req.GoodreadsID,
		HardcoverID:     req.HardcoverID,
		GoogleBooksID:   req.GoogleBooksID,
		PublicationDate: req.PublicationDate,
		Publisher:       req.Publisher,
		Language:        req.Language,
		CoverImageURL:   req.CoverImageURL,
	})
	if handleUpdateErr(r.Context(), w, err, nil, nil, "book", "book", id) {
		return
	}

	dto, err := h.loadBookDTO(r.Context(), b)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to build book DTO",
			slog.String(otelkeys.BookID, b.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update book")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionBookUpdated, "book", b.ID, map[string]any{"title": b.Title})

	writeJSON(r.Context(), w, http.StatusOK, dto)
}

// deleteBook permanently removes a book record.
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
	deleteResource(h.DB, w, r, id, "book", "book", otelkeys.BookID,
		h.DB.GetBook, h.DB.DeleteBook,
		db.AuditActionBookDeleted,
		func(b *db.Book) map[string]any { return map[string]any{"title": b.Title} },
	)
}
