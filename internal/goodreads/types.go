package goodreads

// BookResult holds the normalized book metadata returned by Goodreads lookups
// and searches. Fields are populated on a best-effort basis — some may be
// empty depending on what Goodreads returns for a given title.
type BookResult struct {
	WorkID                string `json:"work_id"`
	WorkLegacyID          int64  `json:"work_legacy_id"`
	BookID                string `json:"book_id"`
	BookLegacyID          int64  `json:"book_legacy_id"`
	BookImageURL          string `json:"book_image_url"`
	BookTitle             string `json:"book_title"`
	BookASIN              string `json:"book_asin"`
	BookISBN              string `json:"book_isbn"`
	BookISBN13            string `json:"book_isbn13"`
	BookLanguage          string `json:"book_language"`
	AuthorID              string `json:"author_id"`
	AuthorName            string `json:"author_name"`
	AuthorLegacyID        int64  `json:"author_legacy_id"`
	AuthorProfileImageURL string `json:"author_profile_image_url"`
}
