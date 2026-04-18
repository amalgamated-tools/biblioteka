import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";
import type { PaginatedBooks } from "../../types";
import BookList from "./BookList.svelte";

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  LayoutGrid: () => {},
  List: () => {},
  ChevronLeft: () => {},
  ChevronRight: () => {},
  Mail: () => {},
}));

const emptyBooks: PaginatedBooks = {
  books: [],
  total: 0,
  limit: 24,
  offset: 0,
};

const fakeBooks: PaginatedBooks = {
  books: [
    {
      id: "b1",
      title: "Test Book",
      description: null,
      asin: null,
      isbn10: null,
      isbn13: null,
      goodreads_id: null,
      hardcover_id: null,
      google_books_id: null,
      publication_date: null,
      publisher: null,
      language: null,
      cover_image_url: null,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
  total: 1,
  limit: 24,
  offset: 0,
};

describe("BookList loading state", () => {
  afterEach(() => cleanup());

  it("exposes 'Loading books...' via role='status' while loading", () => {
    // fetchBooks never resolves, so the component stays in the loading state
    const fetchBooks = vi.fn().mockReturnValue(new Promise(() => {}));
    render(BookList, { props: { fetchBooks } });

    const status = screen.getByRole("status");
    expect(status).toHaveTextContent("Loading books...");
  });

  it("passes the query prop to fetchBooks", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks, query: "tolkien" } });
    await tick();
    await tick();

    expect(fetchBooks).toHaveBeenCalledWith(24, 0, "tolkien");
  });

  it("resets offset to 0 when query changes", async () => {
    // Return enough books to have multiple pages
    const page1: PaginatedBooks = {
      books: Array.from({ length: 2 }, (_, i) => ({
        id: `b${i}`,
        title: `Book ${i}`,
        description: null,
        asin: null,
        isbn10: null,
        isbn13: null,
        goodreads_id: null,
        hardcover_id: null,
        google_books_id: null,
        publication_date: null,
        publisher: null,
        language: null,
        cover_image_url: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      })),
      total: 50,
      limit: 2,
      offset: 0,
    };
    const page2: PaginatedBooks = { ...page1, offset: 2 };
    const searchResult: PaginatedBooks = {
      books: [page1.books[0]],
      total: 1,
      limit: 2,
      offset: 0,
    };

    const fetchBooks = vi
      .fn()
      .mockResolvedValueOnce(page1) // initial load
      .mockResolvedValueOnce(page2) // after next page
      .mockResolvedValueOnce(searchResult); // after query change

    const { rerender } = render(BookList, {
      props: { fetchBooks, pageSize: 2 },
    });
    await tick();
    await tick();

    // Navigate to page 2
    const nextButton = screen.getByRole("button", { name: /Next page/ });
    await fireEvent.click(nextButton);
    await tick();
    await tick();

    // Verify we're on page 2 (offset=2)
    expect(fetchBooks).toHaveBeenLastCalledWith(2, 2, undefined);

    // Change the query prop
    await rerender({ fetchBooks, pageSize: 2, query: "tolkien" });
    await tick();
    await tick();

    // Offset should have reset to 0
    expect(fetchBooks).toHaveBeenLastCalledWith(2, 0, "tolkien");
  });
});

describe("BookList table view link accessibility (WCAG 2.1.1)", () => {
  afterEach(() => cleanup());
  beforeEach(() => {
    window.location.hash = "";
  });

  it("table rows are not interactive", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const row = screen.getByRole("row", { name: /Test Book/i });
    expect(row.tagName).toBe("TR");
    expect(row).not.toHaveAttribute("tabindex");
    expect(row).not.toHaveAttribute("aria-label");
  });

  it("title link remains in the natural tab order", async () => {
    const user = userEvent.setup();
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const titleLink = screen.getByRole("link", { name: "Test Book" });
    expect(titleLink).not.toHaveAttribute("tabindex");

    // Bound the number of tab presses so the test fails fast if focus never reaches the link
    const maxTabPresses = 10;
    let tabPresses = 1;
    await user.tab();
    while (document.activeElement !== titleLink && tabPresses < maxTabPresses) {
      await user.tab();
      tabPresses += 1;
    }
    expect(
      document.activeElement,
      `Expected title link to be reachable within ${maxTabPresses} Tab presses`,
    ).toBe(titleLink);
  });

  it("title link points to the book route", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const titleLink = screen.getByRole("link", { name: "Test Book" });
    expect(titleLink).toHaveAttribute("href", "#books/b1");
  });
});

describe("BookList table view accessibility", () => {
  afterEach(() => cleanup());

  it("labels the book table with aria-label (WCAG 1.3.1)", async () => {
    const manyBooks: PaginatedBooks = {
      books: Array.from({ length: 2 }, (_, i) => ({
        id: `b${i}`,
        title: `Book ${i}`,
        description: null,
        asin: null,
        isbn10: null,
        isbn13: null,
        goodreads_id: null,
        hardcover_id: null,
        google_books_id: null,
        publication_date: null,
        publisher: null,
        language: null,
        cover_image_url: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      })),
      total: 2,
      limit: 24,
      offset: 0,
    };
    const fetchBooks = vi.fn().mockResolvedValue(manyBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    // Switch to table view
    const tableViewButton = screen.getByRole("button", {
      name: "Table view",
    });
    await fireEvent.click(tableViewButton);
    await tick();

    expect(screen.getByRole("table", { name: "Books" })).toBeInTheDocument();
  });
});

describe("BookList pagination accessibility", () => {
  afterEach(() => cleanup());

  it("pagination counter text updates when navigating pages (WCAG 4.1.3)", async () => {
    const makePageBooks = (offset: number): PaginatedBooks => ({
      books: Array.from({ length: 2 }, (_, i) => ({
        id: `b${offset + i}`,
        title: `Book ${offset + i}`,
        description: null,
        asin: null,
        isbn10: null,
        isbn13: null,
        goodreads_id: null,
        hardcover_id: null,
        google_books_id: null,
        publication_date: null,
        publisher: null,
        language: null,
        cover_image_url: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      })),
      total: 50,
      limit: 2,
      offset,
    });

    const fetchBooks = vi
      .fn()
      .mockImplementation((_size: number, off: number) =>
        Promise.resolve(makePageBooks(off)),
      );
    render(BookList, { props: { fetchBooks, pageSize: 2 } });
    await tick();
    await tick();

    expect(screen.getByText(/Page 1 of 25/)).toBeInTheDocument();

    const nextButton = screen.getByRole("button", { name: /Next page/ });
    await fireEvent.click(nextButton);
    await tick();
    await tick();

    expect(screen.getByText(/Page 2 of 25/)).toBeInTheDocument();
  });

  it("toolbar live region announces range updates on page change (WCAG 4.1.3)", async () => {
    const makePageBooks = (offset: number): PaginatedBooks => ({
      books: Array.from({ length: 2 }, (_, i) => ({
        id: `b${offset + i}`,
        title: `Book ${offset + i}`,
        description: null,
        asin: null,
        isbn10: null,
        isbn13: null,
        goodreads_id: null,
        hardcover_id: null,
        google_books_id: null,
        publication_date: null,
        publisher: null,
        language: null,
        cover_image_url: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      })),
      total: 50,
      limit: 2,
      offset,
    });

    const fetchBooks = vi
      .fn()
      .mockImplementation((_size: number, off: number) =>
        Promise.resolve(makePageBooks(off)),
      );
    render(BookList, { props: { fetchBooks, pageSize: 2 } });
    await tick();
    await tick();

    const rangeRegion = screen.getByText(/Showing 1–2 of 50 books/);
    expect(rangeRegion).toHaveAttribute("aria-live", "polite");
    expect(rangeRegion).toHaveAttribute("aria-atomic", "true");

    const nextButton = screen.getByRole("button", { name: /Next page/ });
    await fireEvent.click(nextButton);
    await tick();
    await tick();

    expect(screen.getByText(/Showing 3–4 of 50 books/)).toBeInTheDocument();
  });

  it("pagination buttons have descriptive aria-label (WCAG 2.4.6)", async () => {
    const manyBooks: PaginatedBooks = {
      books: Array.from({ length: 2 }, (_, i) => ({
        id: `b${i}`,
        title: `Book ${i}`,
        description: null,
        asin: null,
        isbn10: null,
        isbn13: null,
        goodreads_id: null,
        hardcover_id: null,
        google_books_id: null,
        publication_date: null,
        publisher: null,
        language: null,
        cover_image_url: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      })),
      total: 50,
      limit: 2,
      offset: 0,
    };
    const fetchBooks = vi.fn().mockResolvedValue(manyBooks);
    render(BookList, { props: { fetchBooks, pageSize: 2 } });
    await tick();
    await tick();

    const prevButton = screen.getByRole("button", {
      name: /Previous page/,
    });
    expect(prevButton).toHaveAttribute(
      "aria-label",
      "Previous page, page 1 of 25",
    );

    const nextButton = screen.getByRole("button", {
      name: /Next page/,
    });
    expect(nextButton).toHaveAttribute("aria-label", "Next page, page 2 of 25");
  });
});

describe("BookList empty state", () => {
  afterEach(() => cleanup());

  it("shows 'No books yet.' when no pollingInterval is set", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(emptyBooks);
    const { container } = render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    expect(container.textContent).toContain("No books yet.");
    expect(container.textContent).not.toContain("Scanning library...");
  });

  it("shows 'No books found.' when a query is set but no results", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(emptyBooks);
    const { container } = render(BookList, {
      props: { fetchBooks, query: "tolkien" },
    });
    await tick();
    await tick();

    expect(container.textContent).toContain("No books found.");
    expect(container.textContent).toContain("Try a different search term.");
    expect(container.textContent).not.toContain("No books yet.");
  });

  it("shows 'Scanning library...' when pollingInterval is set and no books found", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(emptyBooks);
    const { container } = render(BookList, {
      props: { fetchBooks, pollingInterval: 3000 },
    });
    await tick();
    await tick();

    expect(container.textContent).toContain("Scanning library...");
    expect(container.textContent).not.toContain("No books yet.");
  });

  it("shows custom emptyMessage when provided", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(emptyBooks);
    const { container } = render(BookList, {
      props: {
        fetchBooks,
        emptyMessage: 'No results for "tolkien"',
        emptySubMessage: "Try a different search term.",
      },
    });
    await tick();
    await tick();

    expect(container.textContent).toContain('No results for "tolkien"');
    expect(container.textContent).toContain("Try a different search term.");
    expect(container.textContent).not.toContain("No books yet.");
  });
});

describe("BookList polling", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("polls at the specified interval when total is 0", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(emptyBooks);
    render(BookList, { props: { fetchBooks, pollingInterval: 1000 } });
    await tick();
    await tick();

    // Initial load
    expect(fetchBooks).toHaveBeenCalledTimes(1);

    // Advance time to trigger one poll
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchBooks).toHaveBeenCalledTimes(2);

    // Advance time to trigger another poll
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchBooks).toHaveBeenCalledTimes(3);
  });

  it("stops polling once books are found", async () => {
    // First call returns empty, second returns books
    const fetchBooks = vi
      .fn()
      .mockResolvedValueOnce(emptyBooks)
      .mockResolvedValueOnce(fakeBooks);

    render(BookList, { props: { fetchBooks, pollingInterval: 1000 } });
    await tick();
    await tick();

    // Initial load returned empty, so polling starts
    expect(fetchBooks).toHaveBeenCalledTimes(1);

    // Advance time; poll fires and returns books
    await vi.advanceTimersByTimeAsync(1000);
    await tick();
    expect(fetchBooks).toHaveBeenCalledTimes(2);

    // Advance time again; polling should have stopped since books were found
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchBooks).toHaveBeenCalledTimes(2);
  });

  it("calls onBooksFound when books appear for the first time", async () => {
    const fetchBooks = vi
      .fn()
      .mockResolvedValueOnce(emptyBooks)
      .mockResolvedValueOnce(fakeBooks);
    const onBooksFound = vi.fn();

    render(BookList, {
      props: { fetchBooks, pollingInterval: 1000, onBooksFound },
    });
    await tick();
    await tick();

    expect(onBooksFound).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1000);
    await tick();

    expect(onBooksFound).toHaveBeenCalledTimes(1);
  });

  it("does not poll when no pollingInterval is set", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(emptyBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    expect(fetchBooks).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(5000);
    expect(fetchBooks).toHaveBeenCalledTimes(1);
  });
});
