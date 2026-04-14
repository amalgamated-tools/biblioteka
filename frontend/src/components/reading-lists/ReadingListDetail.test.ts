import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  listReadingListBooks: vi.fn(),
}));

vi.mock("../ui/BookList.svelte", () => ({
  default: () => null,
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
}));

import type { ReadingList } from "../../types";
import { readingListStore } from "../../stores/reading-lists.svelte";
import ReadingListDetail from "./ReadingListDetail.svelte";

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: null,
  book_count: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("ReadingListDetail delete confirmation accessibility", () => {
  beforeEach(() => {
    (readingListStore as any).loaded = true;
    (readingListStore as any).loading = false;
    (readingListStore as any).loadError = null;
    (readingListStore as any).lists = [fakeList];
    (readingListStore as any).update = vi.fn().mockResolvedValue(fakeList);
  });

  afterEach(() => {
    (readingListStore as any).loaded = false;
    (readingListStore as any).loading = false;
    (readingListStore as any).loadError = null;
    (readingListStore as any).lists = [];
    cleanup();
    vi.restoreAllMocks();
  });

  it("moves focus to the confirm button when inline delete confirmation appears", async () => {
    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    await fireEvent.click(screen.getByRole("button", { name: "Delete" }));

    const confirmButton = await screen.findByRole("button", {
      name: "Yes, delete",
    });

    await waitFor(() => {
      expect(confirmButton).toHaveFocus();
    });
  });
});
