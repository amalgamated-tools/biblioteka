export interface AIEnrichment {
  id: string;
  book_id: string | null;
  status: "pending" | "applied" | "rejected";
  provider: string;
  model: string;
  suggested_tags: string[];
  reading_level: string | null;
  generated_description: string | null;
  created_at: string;
  updated_at: string;
}

export interface RemoteMetadata {
  id: string;
  book_id: string | null;
  status: string;
  source: string;
  title: string | null;
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
  author_name: string | null;
  created_at: string;
  updated_at: string;
}

export interface MetadataProgressEvent {
  event: "progress" | "complete" | "not_found" | "error";
  source?: string;
  step?: string;
  message?: string;
  metadata_id?: string;
  ai_enrichment_id?: string;
}

export interface MetadataFetchResponse {
  task_id?: string;
  status: "enqueued" | "already_exists" | "already_running";
}

export interface CurrentValues {
  title: string;
  description: string | null;
  publisher: string | null;
  language: string | null;
  publication_date: string | null;
  isbn13: string | null;
  isbn10: string | null;
  asin: string | null;
  goodreads_id: string | null;
  hardcover_id: string | null;
  google_books_id: string | null;
  cover_image_url: string | null;
}
