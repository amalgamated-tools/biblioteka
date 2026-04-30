export const LIBRARY_ORGANIZATION_TYPES = {
  BOOK_PER_FOLDER: "book_per_folder",
  BOOK_PER_FILE: "book_per_file",
  NONE: "none",
} as const;

export type LibraryOrganizationType =
  (typeof LIBRARY_ORGANIZATION_TYPES)[keyof typeof LIBRARY_ORGANIZATION_TYPES];

export const LIBRARY_ORGANIZATION_OPTIONS = [
  {
    value: LIBRARY_ORGANIZATION_TYPES.BOOK_PER_FOLDER,
    label: "Book Per Folder (Author/Title/file)",
  },
  {
    value: LIBRARY_ORGANIZATION_TYPES.BOOK_PER_FILE,
    label: "Multiple Books Per Author (Author/files)",
  },
  {
    value: LIBRARY_ORGANIZATION_TYPES.NONE,
    label: "No Organization",
  },
] as const;

export interface Library {
  id: string;
  name: string;
  paths: string[];
  organization_type: LibraryOrganizationType;
  monitored: boolean;
  created_at: string;
  updated_at: string;
}

export interface LibraryInput {
  name: string;
  paths: string[];
  organization_type: LibraryOrganizationType;
  monitored: boolean;
}
