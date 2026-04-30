export interface BookFile {
  id: string;
  book_id: string;
  file_type: string;
  file_name: string;
  file_size: number;
  file_hash: string | null;
  file_path: string;
  download_count: number;
  created_at: string;
  updated_at: string;
}

export interface BookFileInput {
  file_type: string;
  file_name: string;
  file_size: number;
  file_hash?: string | null;
  file_path: string;
}

export interface BookSummary {
  id: string;
  title: string;
  description: string | null;
  asin: string | null;
  isbn10: string | null;
  isbn13: string | null;
  goodreads_id: string | null;
  hardcover_id: string | null;
  google_books_id: string | null;
  publication_date: string | null;
  publisher: string | null;
  language: string | null;
  cover_image_url: string | null;
  created_at: string;
  updated_at: string;
}

export interface Author {
  id: string;
  name: string;
  goodreads_id: string | null;
  hardcover_id: string | null;
  google_books_id: string | null;
  image_url: string | null;
  created_at: string;
  updated_at: string;
}

export interface AuthorInput {
  name: string;
  goodreads_id?: string | null;
  hardcover_id?: string | null;
  google_books_id?: string | null;
  image_url?: string | null;
}

export interface Series {
  id: string;
  name: string;
  goodreads_id: string | null;
  hardcover_id: string | null;
  google_books_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface SeriesInput {
  name: string;
  goodreads_id?: string | null;
  hardcover_id?: string | null;
  google_books_id?: string | null;
}

export interface BookSeriesEntry {
  series: Series;
  position: number | null;
}

export interface Tag {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface TagInput {
  name: string;
}

export interface Book extends BookSummary {
  authors: Author[];
  series: BookSeriesEntry[];
  tags: Tag[];
  files: BookFile[];
}

export interface BookInput {
  title: string;
  description?: string | null;
  asin?: string | null;
  isbn10?: string | null;
  isbn13?: string | null;
  goodreads_id?: string | null;
  hardcover_id?: string | null;
  google_books_id?: string | null;
  publication_date?: string | null;
  publisher?: string | null;
  language?: string | null;
  cover_image_url?: string | null;
}

export interface PaginatedBooks {
  books: BookSummary[];
  total: number;
  limit: number;
  offset: number;
}
