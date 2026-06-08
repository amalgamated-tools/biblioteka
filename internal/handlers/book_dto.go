package handlers

import (
	"context"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

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
	CoverImageURL   *string `json:"cover_image_url"`
}

type bookSeriesEntryDTO struct {
	Series   seriesDTO `json:"series"`
	Position *float64  `json:"position"`
}

type bookFileDTO struct {
	ID            string       `json:"id"`
	BookID        string       `json:"book_id"`
	FileType      string       `json:"file_type"`
	FileName      string       `json:"file_name"`
	FileSize      int64        `json:"file_size"`
	FileHash      *string      `json:"file_hash"`
	FilePath      string       `json:"file_path"`
	DownloadCount int64        `json:"download_count"`
	CreatedAt     db.Timestamp `json:"created_at"`
	UpdatedAt     db.Timestamp `json:"updated_at"`
}

func toBookFileDTO(bf *db.BookFile) bookFileDTO {
	return bookFileDTO{
		ID:            bf.ID,
		BookID:        bf.BookID,
		FileType:      bf.FileType,
		FileName:      bf.FileName,
		FileSize:      bf.FileSize,
		FileHash:      bf.FileHash,
		FilePath:      bf.FilePath,
		DownloadCount: bf.DownloadCount,
		CreatedAt:     bf.CreatedAt,
		UpdatedAt:     bf.UpdatedAt,
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
	CoverImageURL   *string      `json:"cover_image_url"`
	CreatedAt       db.Timestamp `json:"created_at"`
	UpdatedAt       db.Timestamp `json:"updated_at"`
}

type bookListDTO struct {
	Books []bookSummaryDTO `json:"books"`
	paginationMeta
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
	CoverImageURL   *string              `json:"cover_image_url"`
	Authors         []authorDTO          `json:"authors"`
	Series          []bookSeriesEntryDTO `json:"series"`
	Tags            []tagDTO             `json:"tags"`
	Files           []bookFileDTO        `json:"files"`
	CreatedAt       db.Timestamp         `json:"created_at"`
	UpdatedAt       db.Timestamp         `json:"updated_at"`
}

// loadBookDTO builds a bookDTO for b by loading all related entities via
// [db.DB.LoadBookRelations] (authors, series, tags, and files in one call).
func (h *BookHandler) loadBookDTO(ctx context.Context, b *db.Book) (bookDTO, error) {
	rels, err := h.DB.LoadBookRelations(ctx, b.ID)
	if err != nil {
		return bookDTO{}, err
	}

	return bookDTO{
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
		CoverImageURL:   b.CoverImageURL,
		Authors:         mapSlice(rels.Authors, toAuthorDTO),
		Series:          mapSlice(rels.Series, toBookSeriesEntryDTO),
		Tags:            mapSlice(rels.Tags, toTagDTO),
		Files:           mapSlice(rels.Files, toBookFileDTO),
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}, nil
}
