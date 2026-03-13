export interface User {
  id: string;
  email: string;
  oidc_linked: boolean;
  is_admin: boolean;
}

export type ArrServiceType = "radarr" | "sonarr" | "prowlarr" | "seerr";

export interface ArrService {
  id: string;
  name: string;
  type: ArrServiceType;
  url: string;
  api_key: string;
  created_at: string;
  updated_at: string;
}

export interface ArrServiceInput {
  name: string;
  type: ArrServiceType;
  url: string;
  api_key: string;
}

export interface Library {
  id: string;
  name: string;
  paths: string[];
  organization_type: string;
  monitored: boolean;
  created_at: string;
  updated_at: string;
}

export interface LibraryInput {
  name: string;
  paths: string[];
  organization_type: string;
  monitored: boolean;
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
  goodreads_id?: string;
  hardcover_id?: string;
  google_books_id?: string;
  image_url?: string;
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
  goodreads_id?: string;
  hardcover_id?: string;
  google_books_id?: string;
}

export interface BookSeriesEntry {
  series: Series;
  position: number | null;
}

export interface BookFile {
  id: string;
  book_id: string;
  file_type: string;
  file_name: string;
  file_size: number;
  file_hash: string | null;
  file_path: string;
  created_at: string;
  updated_at: string;
}

export interface BookFileInput {
  file_type: string;
  file_name: string;
  file_size: number;
  file_hash?: string;
  file_path: string;
}

export interface Book {
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
  num_pages: number | null;
  cover_image_url: string | null;
  authors: Author[];
  series: BookSeriesEntry[];
  files: BookFile[];
  created_at: string;
  updated_at: string;
}

export interface BookInput {
  title: string;
  description?: string;
  asin?: string;
  isbn10?: string;
  isbn13?: string;
  goodreads_id?: string;
  hardcover_id?: string;
  google_books_id?: string;
  publication_date?: string;
  publisher?: string;
  language?: string;
  num_pages?: number;
  cover_image_url?: string;
}
