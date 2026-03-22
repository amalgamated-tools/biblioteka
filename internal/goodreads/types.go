package goodreads

type SearchResult struct {
	WorkID         string `json:"work_id"`
	WorkLegacyID   int64  `json:"work_legacy_id"`
	BookID         string `json:"book_id"`
	BookLegacyID   int64  `json:"book_legacy_id"`
	Title          string `json:"title"`
	Author         string `json:"author"`
	AuthorID       string `json:"author_id"`
	AuthorLegacyID int64  `json:"author_legacy_id"`
}
