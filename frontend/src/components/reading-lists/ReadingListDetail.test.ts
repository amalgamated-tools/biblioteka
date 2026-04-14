import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
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
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).loading = false;
    vi.mocked(readingListStore).loadError = null as string | null;
    vi.mocked(readingListStore).lists = [fakeList] as ReadingList[];
    vi.mocked(readingListStore).update = vi.fn().mockResolvedValue(fakeList);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("moves focus to the confirm button when inline delete confirmation appears", async () => {
    const user = userEvent.setup();
    render(ReadingListDetail, { props: { listId: fakeList.id } });
    await tick();

    const deleteButton = screen.getByRole("button", { name: "Delete" });
    await user.click(deleteButton);

    const confirmButton = await screen.findByRole("button", {
      name: "Yes, delete",
    });

    await waitFor(() => {
      expect(confirmButton).toHaveFocus();
    });
  });
});
