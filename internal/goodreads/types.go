package goodreads

type SearchResult struct {
	WorkID                string `json:"work_id"`
	WorkLegacyID          int64  `json:"work_legacy_id"`
	BookID                string `json:"book_id"`
	BookLegacyID          int64  `json:"book_legacy_id"`
	BookImageURL          string `json:"book_image_url"`
	Title                 string `json:"title"`
	AuthorID              string `json:"author_id"`
	AuthorName            string `json:"author_name"`
	AuthorLegacyID        int64  `json:"author_legacy_id"`
	AuthorProfileImageURL string `json:"author_profile_image_url"`
}
