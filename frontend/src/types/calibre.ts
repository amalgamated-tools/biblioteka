export interface CalibrePreviewSeriesEntry {
  name: string;
  position: number;
}

export interface CalibrePreviewBook {
  calibre_id: number;
  title: string;
  authors: string[];
  series: CalibrePreviewSeriesEntry[];
  publisher?: string;
  publication_date?: string;
  isbn13?: string;
  isbn10?: string;
  formats: string[];
}

export interface CalibrePreview {
  total: number;
  books: CalibrePreviewBook[];
}

export interface CalibreImportResult {
  total: number;
  imported: number;
  skipped: number;
  errors: number;
}
