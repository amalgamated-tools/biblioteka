import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/reading-lists.svelte", () => ({
  readingListStore: {
    loaded: false,
    loading: false,
    loadError: null as string | null,
    lists: [] as Array<{
      id: string;
      name: string;
      description: string | null;
      book_count: number;
      created_at: string;
      updated_at: string;
    }>,
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

vi.mock("../../lib/api", () => ({
  listReadingListBooks: vi.fn().mockResolvedValue({
    books: [],
    total: 0,
    limit: 24,
    offset: 0,
  }),
}));

vi.mock("../ui/AlertBanner.svelte", () => ({
  default: () => null,
}));

vi.mock("../ui/BookList.svelte", () => ({
  default: () => null,
}));

vi.mock("lucide-svelte", () => ({
  ArrowLeft: () => null,
  Pencil: () => null,
  Trash2: () => null,
  BookMarked: () => null,
  X: () => null,
  Check: () => null,
}));

import ReadingListDetail from "./ReadingListDetail.svelte";
import { readingListStore } from "../../stores/reading-lists.svelte";

describe("ReadingListDetail accessibility", () => {
  beforeEach(() => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).loading = false;
    vi.mocked(readingListStore).loadError = null;
    vi.mocked(readingListStore).lists = [
      {
        id: "rl-1",
        name: "Test List",
        description: null,
        book_count: 0,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("marks the edit list name input as aria-required", async () => {
    render(ReadingListDetail, { props: { listId: "rl-1" } });
    await tick();

    // Click the edit button to enter edit mode
    const editButton = screen.getByRole("button", { name: /Edit/i });
    await fireEvent.click(editButton);
    await tick();

    // Verify the name input has aria-required
    const nameInput = screen.getByDisplayValue("Test List");
    expect(nameInput).toHaveAttribute("aria-required", "true");
  });
});
