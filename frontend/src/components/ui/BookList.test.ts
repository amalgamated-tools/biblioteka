import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, cleanup } from "@testing-library/svelte";
import { tick } from "svelte";
import type { PaginatedBooks } from "../../types";
import BookList from "./BookList.svelte";

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  LayoutGrid: () => {},
  List: () => {},
  ChevronLeft: () => {},
  ChevronRight: () => {},
}));

const emptyBooks: PaginatedBooks = { books: [], total: 0, limit: 24, offset: 0 };

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
