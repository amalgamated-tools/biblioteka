import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/svelte";
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
});

describe("BookList table view keyboard accessibility (WCAG 2.1.1)", () => {
  afterEach(() => cleanup());

  it("table rows have tabindex=0 and role=link", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const row = screen.getByRole("link", { name: "View Test Book" });
    expect(row.tagName).toBe("TR");
    expect(row).toHaveAttribute("tabindex", "0");
  });

  it("table rows navigate on Enter key", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const row = screen.getByRole("link", { name: "View Test Book" });
    await fireEvent.keyDown(row, { key: "Enter" });

    expect(window.location.hash).toBe("#books/b1");
  });

  it("table rows navigate on Space key", async () => {
    window.location.hash = "";
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const row = screen.getByRole("link", { name: "View Test Book" });
    await fireEvent.keyDown(row, { key: " " });

    expect(window.location.hash).toBe("#books/b1");
  });

  it("title anchor has tabindex=-1 to avoid double-tabbing", async () => {
    const fetchBooks = vi.fn().mockResolvedValue(fakeBooks);
    const { container } = render(BookList, { props: { fetchBooks } });
    await tick();
    await tick();

    const tableViewButton = screen.getByRole("button", { name: "Table view" });
    await fireEvent.click(tableViewButton);
    await tick();

    const titleLink = container.querySelector(`a[href="#books/b1"]`);
    expect(titleLink).toHaveAttribute("tabindex", "-1");
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

  it("pagination counter has aria-atomic but no duplicate aria-live (WCAG 4.1.3)", async () => {
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

    const pageCounter = screen.getByText(/Page 1 of/);
    expect(pageCounter).not.toHaveAttribute("aria-live");
    expect(pageCounter).toHaveAttribute("aria-atomic", "true");
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
