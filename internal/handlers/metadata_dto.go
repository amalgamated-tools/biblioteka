package handlers

import "github.com/amalgamated-tools/biblioteka/internal/db"

type metadataDTO struct {
	ID              string       `json:"id"`
	BookID          *string      `json:"book_id"`
	Status          string       `json:"status"`
	Source          string       `json:"source"`
	Title           *string      `json:"title"`
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
	AuthorName      *string      `json:"author_name"`
	CreatedAt       db.Timestamp `json:"created_at"`
	UpdatedAt       db.Timestamp `json:"updated_at"`
}

// Intentionally omitted from the DTO: AuthorGoodreadsID, AuthorImageURL,
// GoodreadsWorkID, GoodreadsBookLegacyID, GoodreadsWorkLegacyID, and
// GoodreadsAuthorLegacyID. These are internal Goodreads identifiers used
// only during enrichment and have no value in the user-facing API.
func toMetadataDTO(gm *db.GoodreadsMetadata) metadataDTO {
	return metadataDTO{
		ID:              gm.ID,
		BookID:          gm.BookID,
		Status:          gm.Status,
		Source:          db.MetadataSourceGoodreads,
		Title:           gm.Title,
		Description:     gm.Description,
		ASIN:            gm.ASIN,
		ISBN10:          gm.ISBN10,
		ISBN13:          gm.ISBN13,
		GoodreadsID:     gm.GoodreadsID,
		HardcoverID:     gm.HardcoverID,
		GoogleBooksID:   gm.GoogleBooksID,
		PublicationDate: gm.PublicationDate,
		Publisher:       gm.Publisher,
		Language:        gm.Language,
		CoverImageURL:   gm.CoverImageURL,
		AuthorName:      gm.AuthorName,
		CreatedAt:       gm.CreatedAt,
		UpdatedAt:       gm.UpdatedAt,
	}
}

type fetchMetadataResponse struct {
	TaskID string `json:"task_id,omitempty"`
	Status string `json:"status"`
}
