import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import type { Book } from "../../types";

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  ArrowLeft: () => {},
  Download: () => {},
  Pencil: () => {},
  FileText: () => {},
  User: () => {},
  BookMarked: () => {},
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: { navigate: vi.fn() },
}));

const mockGetBook = vi.fn();
const mockBookFileDownloadUrl = vi.fn(
  (id: string) => `/api/book-files/${id}/download`,
);
vi.mock("../../lib/api", () => ({
  getBook: (...args: unknown[]) => mockGetBook(...args),
  bookFileDownloadUrl: (...args: [string]) => mockBookFileDownloadUrl(...args),
}));

import BookDetail from "./BookDetail.svelte";

const fakeBook: Book = {
  id: "b1",
  title: "The Hobbit",
  description: "A fantasy novel about a hobbit.",
  asin: null,
  isbn10: null,
  isbn13: "9780547928227",
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: "1937-09-21",
  publisher: "Allen & Unwin",
  language: "en",
  cover_image_url: null,
  authors: [
    {
      id: "a1",
      name: "J.R.R. Tolkien",
      goodreads_id: null,
      hardcover_id: null,
      google_books_id: null,
      image_url: null,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
  series: [],
  files: [],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeBookWithFiles: Book = {
  ...fakeBook,
  files: [
    {
      id: "f1",
      book_id: "b1",
      file_type: "epub",
      file_name: "the-hobbit.epub",
      file_size: 12345,
      file_hash: null,
      file_path: "/library/the-hobbit.epub",
      download_count: 0,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
    {
      id: "f2",
      book_id: "b1",
      file_type: "pdf",
      file_name: "the-hobbit.pdf",
      file_size: 56789,
      file_hash: null,
      file_path: "/library/the-hobbit.pdf",
      download_count: 0,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
};

describe("BookDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBookFileDownloadUrl.mockImplementation(
      (id: string) => `/api/book-files/${id}/download`,
    );
  });

  afterEach(() => cleanup());

  it("shows loading state initially", () => {
    mockGetBook.mockReturnValue(new Promise(() => {}));
    render(BookDetail, { bookId: "b1" });
    expect(screen.getByText("Loading book...")).toBeInTheDocument();
  });

  it("renders book title after loading", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
        "The Hobbit",
      );
    });
  });

  it("renders book details fields", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Allen & Unwin")).toBeInTheDocument();
    });
    expect(screen.getByText("9780547928227")).toBeInTheDocument();
    expect(screen.getByText("1937-09-21")).toBeInTheDocument();
  });

  it("renders the description section", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByText("A fantasy novel about a hobbit."),
      ).toBeInTheDocument();
    });
  });

  it("renders authors list", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("J.R.R. Tolkien")).toBeInTheDocument();
    });
  });

  it("shows error message when loading fails", async () => {
    mockGetBook.mockRejectedValue(new Error("Not found"));
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Not found")).toBeInTheDocument();
    });
  });

  it("calls getBook with the provided bookId", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(mockGetBook).toHaveBeenCalledWith("b1");
    });
  });

  it("renders descriptive aria-labels for file download links", async () => {
    mockGetBook.mockResolvedValue(fakeBookWithFiles);
    render(BookDetail, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("link", {
          name: "Download the-hobbit.epub (EPUB)",
        }),
      ).toBeInTheDocument();
    });

    expect(
      screen.getByRole("link", { name: "Download the-hobbit.pdf (PDF)" }),
    ).toBeInTheDocument();
  });
});
