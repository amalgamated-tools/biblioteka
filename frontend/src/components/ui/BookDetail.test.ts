import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";

vi.mock("lucide-svelte", () => ({
  ArrowLeft: () => {},
  BookOpen: () => {},
  Download: () => {},
  FileText: () => {},
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("../../lib/api", () => ({
  getBook: vi.fn(),
  bookFileDownloadUrl: (id: string) => `/api/book-files/${id}/download`,
}));

import BookDetail from "./BookDetail.svelte";
import { routerStore } from "../../stores/router.svelte";
import { getBook } from "../../lib/api";
import type { Book } from "../../types";

const mockBook: Book = {
  id: "b1",
  title: "The Hobbit",
  description: "A fantasy novel.",
  asin: null,
  isbn10: null,
  isbn13: null,
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: "1937-09-21",
  publisher: "Allen & Unwin",
  language: "English",
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
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
  files: [
    {
      id: "f1",
      book_id: "b1",
      file_type: "epub",
      file_name: "the-hobbit.epub",
      file_size: 1048576,
      file_hash: null,
      file_path: "/books/the-hobbit.epub",
      download_count: 5,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
};

describe("BookDetail", () => {
  afterEach(() => {
    cleanup();
    vi.mocked(routerStore.navigate).mockClear();
    vi.mocked(getBook).mockReset();
  });

  it("shows loading state initially", () => {
    vi.mocked(getBook).mockReturnValue(new Promise(() => {}));
    render(BookDetail, { bookId: "b1" });
    expect(screen.getByRole("status")).toHaveTextContent("Loading book...");
  });

  it("renders book title after loading", async () => {
    vi.mocked(getBook).mockResolvedValue(mockBook);
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
        "The Hobbit",
      );
    });
  });

  it("renders the author name", async () => {
    vi.mocked(getBook).mockResolvedValue(mockBook);
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByText(/J\.R\.R\. Tolkien/)).toBeInTheDocument();
    });
  });

  it("renders publisher, language, and publication date", async () => {
    vi.mocked(getBook).mockResolvedValue(mockBook);
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByText(/Allen & Unwin/)).toBeInTheDocument();
      expect(screen.getByText(/English/)).toBeInTheDocument();
      expect(screen.getByText(/1937-09-21/)).toBeInTheDocument();
    });
  });

  it("renders the description", async () => {
    vi.mocked(getBook).mockResolvedValue(mockBook);
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByText("A fantasy novel.")).toBeInTheDocument();
    });
  });

  it("renders the file with download button", async () => {
    vi.mocked(getBook).mockResolvedValue(mockBook);
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByText("the-hobbit.epub")).toBeInTheDocument();
      expect(screen.getByText("epub")).toBeInTheDocument();
      expect(screen.getByText("5 downloads")).toBeInTheDocument();
      const downloadLink = screen.getByRole("link", { name: /Download/ });
      expect(downloadLink).toHaveAttribute(
        "href",
        "/api/book-files/f1/download",
      );
    });
  });

  it("renders a back button", () => {
    vi.mocked(getBook).mockReturnValue(new Promise(() => {}));
    render(BookDetail, { bookId: "b1" });
    expect(
      screen.getByRole("button", { name: /Back to books/ }),
    ).toBeInTheDocument();
  });

  it("shows error state when loading fails", async () => {
    vi.mocked(getBook).mockRejectedValue(new Error("Network error"));
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });
  });

  it("shows singular download count", async () => {
    const singleDownloadBook = {
      ...mockBook,
      files: [
        {
          ...mockBook.files[0],
          download_count: 1,
        },
      ],
    };
    vi.mocked(getBook).mockResolvedValue(singleDownloadBook);
    render(BookDetail, { bookId: "b1" });
    await waitFor(() => {
      expect(screen.getByText("1 download")).toBeInTheDocument();
    });
  });
});
