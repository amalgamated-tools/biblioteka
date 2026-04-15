import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  listReadingListBooks: vi.fn(),
}));

vi.mock("../../stores/reading-lists.svelte", () => ({
  readingListStore: {
    loaded: true,
    loading: false,
    loadError: null,
    lists: [],
    load: vi.fn().mockResolvedValue(undefined),
    update: vi.fn(),
    remove: vi.fn().mockResolvedValue(undefined),
  },
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({
  ArrowLeft: () => null,
  Pencil: () => null,
  Trash2: () => null,
  BookMarked: () => null,
  X: () => null,
  Check: () => null,
  LayoutGrid: () => null,
  List: () => null,
  ChevronLeft: () => null,
  ChevronRight: () => null,
  BookOpen: () => null,
  Mail: () => null,
}));

import type { BookSummary, ReadingList } from "../../types";
import { listReadingListBooks } from "../../lib/api";
import { readingListStore } from "../../stores/reading-lists.svelte";
import { routerStore } from "../../stores/router.svelte";
import ReadingListDetail from "./ReadingListDetail.svelte";

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: "A queue of upcoming books",
  book_count: 26,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeBook: BookSummary = {
  id: "b1",
  title: "The Hobbit",
  description: null,
  asin: null,
  isbn10: null,
  isbn13: null,
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: null,
  publisher: "Allen & Unwin",
  language: "en",
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("ReadingListDetail", () => {
  beforeEach(() => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).loading = false;
    vi.mocked(readingListStore).loadError = null as string | null;
    vi.mocked(readingListStore).lists = [fakeList] as ReadingList[];
    vi.mocked(readingListStore).load = vi.fn().mockResolvedValue(undefined);
    vi.mocked(readingListStore).update = vi.fn().mockResolvedValue(fakeList);
    vi.mocked(readingListStore).remove = vi.fn().mockResolvedValue(undefined);
    vi.mocked(listReadingListBooks).mockResolvedValue({
      books: [fakeBook],
      total: 1,
      limit: 24,
      offset: 0,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the reading list from the store", async () => {
    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    expect(
      screen.getByRole("heading", { level: 1, name: fakeList.name }),
    ).toBeInTheDocument();
    expect(screen.getByText("26 books")).toBeInTheDocument();
    expect(screen.getByText(fakeList.description!)).toBeInTheDocument();
  });

  it("loads lists when the store is not loaded", async () => {
    vi.mocked(readingListStore).loaded = false;
    vi.mocked(readingListStore).lists = [];

    render(ReadingListDetail, { props: { listId: fakeList.id } });

    await waitFor(() => {
      expect(readingListStore.load).toHaveBeenCalledTimes(1);
    });
  });

  it("saves edits through the reading list store", async () => {
    const user = userEvent.setup();

    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    await user.click(screen.getByRole("button", { name: "Edit" }));

    const nameInput = screen.getByLabelText(/Name/i);
    const descriptionInput = screen.getByLabelText(/Description/i);

    await user.clear(nameInput);
    await user.type(nameInput, "  Updated List Name  ");
    await user.clear(descriptionInput);
    await user.type(descriptionInput, "  Updated description  ");

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(readingListStore.update).toHaveBeenCalledWith(fakeList.id, {
        name: "Updated List Name",
        description: "Updated description",
      });
    });

    expect(
      screen.queryByRole("heading", { level: 2, name: "Edit Reading List" }),
    ).not.toBeInTheDocument();
  });

  it("cancels editing and restores original values when editing is restarted", async () => {
    const user = userEvent.setup();

    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    await user.click(screen.getByRole("button", { name: "Edit" }));

    const nameInput = screen.getByLabelText(/Name/i);
    await user.clear(nameInput);
    await user.type(nameInput, "Temporary Name");

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.click(screen.getByRole("button", { name: "Edit" }));

    expect(screen.getByLabelText(/Name/i)).toHaveValue(fakeList.name);
  });

  it("shows delete confirmation and deletes the list on confirm", async () => {
    const user = userEvent.setup();

    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByText("Delete this list?")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Yes, delete" }));

    await waitFor(() => {
      expect(readingListStore.remove).toHaveBeenCalledWith(fakeList.id);
      expect(routerStore.navigate).toHaveBeenCalledWith("reading-lists");
    });
  });

  it("surfaces delete API errors in the component error state", async () => {
    const user = userEvent.setup();

    vi.mocked(readingListStore).remove = vi
      .fn()
      .mockRejectedValueOnce(new Error("Delete failed"));

    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    await user.click(screen.getByRole("button", { name: "Delete" }));
    await user.click(screen.getByRole("button", { name: "Yes, delete" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Delete failed");
    expect(screen.queryByRole("button", { name: "Yes, delete" })).toBeNull();
  });

  it("renders paginated books and requests the next page", async () => {
    const user = userEvent.setup();

    vi.mocked(listReadingListBooks).mockImplementation(
      async (_listId: string, limit = 24, offset = 0) => ({
        books: [
          {
            ...fakeBook,
            id: `book-${offset}`,
            title: `Book ${offset + 1}`,
          },
        ],
        total: 50,
        limit,
        offset,
      }),
    );

    render(ReadingListDetail, { props: { listId: fakeList.id } });

    await waitFor(() => {
      expect(listReadingListBooks).toHaveBeenCalledWith(fakeList.id, 24, 0);
      expect(screen.getByText("Page 1 of 3")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /Next page/i }));

    await waitFor(() => {
      expect(listReadingListBooks).toHaveBeenCalledWith(fakeList.id, 24, 24);
      expect(screen.getByText("Page 2 of 3")).toBeInTheDocument();
    });
  });
});
