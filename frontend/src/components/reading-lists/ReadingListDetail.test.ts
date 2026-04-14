import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

const fakeList = {
  id: "rl-1",
  name: "Test List",
  description: null,
  book_count: 0,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

vi.mock("../../stores/reading-lists.svelte", () => ({
  readingListStore: {
    loaded: true,
    loading: false,
    loadError: null,
    lists: [fakeList],
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
  listReadingListBooks: vi.fn(),
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

describe("ReadingListDetail accessibility", () => {
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
