import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";
import type { Book, RemoteMetadata } from "../../types";

vi.mock("lucide-svelte", () => ({
  ArrowLeft: () => {},
  BookOpen: () => {},
  Search: () => {},
  ArrowRight: () => {},
  Check: () => {},
  X: () => {},
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: { navigate: vi.fn() },
}));

const mockGetBook = vi.fn();
const mockUpdateBook = vi.fn();
const mockGetMetadata = vi.fn();
const mockFetchMetadata = vi.fn();
const mockRejectMetadata = vi.fn();
const mockSubscribeToMetadataEvents = vi.fn();

vi.mock("../../lib/api", () => ({
  getBook: (...args: unknown[]) => mockGetBook(...args),
  updateBook: (...args: unknown[]) => mockUpdateBook(...args),
  getMetadata: (...args: unknown[]) => mockGetMetadata(...args),
  fetchMetadata: (...args: unknown[]) => mockFetchMetadata(...args),
  rejectMetadata: (...args: unknown[]) => mockRejectMetadata(...args),
  subscribeToMetadataEvents: (...args: unknown[]) =>
    mockSubscribeToMetadataEvents(...args),
}));

import BookEdit from "./BookEdit.svelte";

const fakeBook: Book = {
  id: "b1",
  title: "The Hobbit",
  description: "A fantasy novel",
  asin: null,
  isbn10: null,
  isbn13: "9780547928227",
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: null,
  publisher: "Allen & Unwin",
  language: "en",
  cover_image_url: null,
  authors: [],
  series: [],
  files: [],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeMetadata: RemoteMetadata = {
  id: "m1",
  book_id: "b1",
  status: "pending",
  source: "goodreads",
  title: "The Hobbit (Updated)",
  description: "Updated description",
  asin: null,
  isbn10: null,
  isbn13: "9780547928227",
  goodreads_id: "5907",
  hardcover_id: null,
  google_books_id: null,
  publication_date: "1937-09-21",
  publisher: "Allen & Unwin",
  language: "en",
  cover_image_url: null,
  author_name: "J.R.R. Tolkien",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("BookEdit", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMetadata.mockRejectedValue(new Error("not found"));
  });

  afterEach(() => cleanup());

  it("shows loading state initially", () => {
    mockGetBook.mockReturnValue(new Promise(() => {}));
    render(BookEdit, { bookId: "b1" });
    expect(screen.getByText("Loading book...")).toBeInTheDocument();
  });

  it("renders the Edit Book heading after loading", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
        "Edit Book",
      );
    });
  });

  it("shows required-fields legend with sr-only asterisk description", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    const { container } = render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      const legend = container.querySelector("form p");
      expect(legend).toBeInTheDocument();
      expect(legend?.textContent).toMatch(/are required/i);
      const visual = legend?.querySelector('span[aria-hidden="true"]');
      expect(visual).toHaveTextContent("*");
      const srOnly = legend?.querySelector(".sr-only");
      expect(srOnly).toHaveTextContent("an asterisk");
    });
  });

  it("populates form fields from the book", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      const titleInput = screen.getByPlaceholderText(
        "Book title",
      ) as HTMLInputElement;
      expect(titleInput.value).toBe("The Hobbit");
    });

    const publisherInput = screen.getByPlaceholderText(
      "Publisher",
    ) as HTMLInputElement;
    expect(publisherInput.value).toBe("Allen & Unwin");
  });

  it("shows error state when book loading fails", async () => {
    mockGetBook.mockRejectedValue(new Error("Failed to load"));
    render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Failed to load")).toBeInTheDocument();
    });
  });

  it("renders the Fetch Metadata button", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Fetch Metadata")).toBeInTheDocument();
    });
  });

  it("renders Save Changes and Cancel buttons", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
    });
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("renders metadata comparison when pending metadata exists", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    mockGetMetadata.mockResolvedValue(fakeMetadata);
    render(BookEdit, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Fetched Metadata")).toBeInTheDocument();
    });
  });

  it("shows form validation error when title is empty", async () => {
    mockGetBook.mockResolvedValue(fakeBook);
    render(BookEdit, { bookId: "b1" });

    const user = userEvent.setup();

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Book title")).toBeInTheDocument();
    });

    const titleInput = screen.getByPlaceholderText("Book title");
    await user.clear(titleInput);
    await user.click(screen.getByText("Save Changes"));

    await waitFor(() => {
      expect(screen.getByText("Title is required")).toBeInTheDocument();
    });
  });
});
