import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../stores/reading-lists.svelte", () => ({
  readingListStore: {
    loaded: false,
    loading: false,
    loadError: null as string | null,
    lists: [],
    load: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    addBook: vi.fn(),
    removeBook: vi.fn(),
  },
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "",
    navigate: vi.fn(),
  },
}));

vi.mock("./ui/AlertBanner.svelte", () => ({ default: () => {} }));
vi.mock("./reading-lists/ReadingListDetail.svelte", () => ({
  default: () => {},
}));

vi.mock("lucide-svelte", () => ({
  BookMarked: () => {},
  Plus: () => {},
  BookOpen: () => {},
  Pencil: () => {},
  Trash2: () => {},
  X: () => {},
  Check: () => {},
}));

import ReadingLists from "./ReadingLists.svelte";
import { readingListStore } from "../stores/reading-lists.svelte";
import { routerStore } from "../stores/router.svelte";
import type { ReadingList } from "../types";

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: null,
  book_count: 3,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("ReadingLists", () => {
  beforeEach(() => {
    vi.mocked(readingListStore).loaded = false;
    vi.mocked(readingListStore).loadError = null;
    vi.mocked(readingListStore).lists = [];
    vi.mocked(routerStore).subPath = "";
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("calls load on mount when not loaded", async () => {
    vi.mocked(readingListStore).loaded = false;
    render(ReadingLists);
    await tick();
    expect(vi.mocked(readingListStore).load).toHaveBeenCalled();
  });

  it("shows empty state when no lists and loaded", async () => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).loadError = null;
    vi.mocked(readingListStore).lists = [];
    render(ReadingLists);
    await tick();

    expect(screen.getByText(/No reading lists yet/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Create Your First List/i }),
    ).toBeInTheDocument();
  });

  it("does not show empty state when load failed", async () => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).loadError = "network error";
    vi.mocked(readingListStore).lists = [];
    render(ReadingLists);
    await tick();

    expect(screen.queryByText(/No reading lists yet/i)).not.toBeInTheDocument();
  });

  it("shows list cards when lists exist", async () => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).lists = [fakeList];
    render(ReadingLists);
    await tick();

    expect(screen.getByText("To Read")).toBeInTheDocument();
    expect(screen.getByText("3 books")).toBeInTheDocument();
  });

  it("shows 'New List' button in header", async () => {
    vi.mocked(readingListStore).loaded = true;
    render(ReadingLists);
    await tick();

    expect(
      screen.getByRole("button", { name: /New List/i }),
    ).toBeInTheDocument();
  });

  it("shows create form after clicking New List", async () => {
    vi.mocked(readingListStore).loaded = true;
    render(ReadingLists);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New List/i }));
    await tick();

    expect(
      screen.getByRole("heading", { name: /Create Reading List/i }),
    ).toBeInTheDocument();
  });

  it("hides New List button when form is open", async () => {
    vi.mocked(readingListStore).loaded = true;
    render(ReadingLists);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New List/i }));
    await tick();

    expect(
      screen.queryByRole("button", { name: /New List/i }),
    ).not.toBeInTheDocument();
  });

  it("marks the create name field invalid after a create error", async () => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(readingListStore).create = vi
      .fn()
      .mockRejectedValueOnce(new Error("Name is required"));
    render(ReadingLists);
    await tick();

    await fireEvent.click(screen.getByRole("button", { name: /New List/i }));
    await fireEvent.input(screen.getByLabelText(/Name/i), {
      target: { value: "My list" },
    });
    await fireEvent.click(screen.getByRole("button", { name: /^Create$/i }));
    await tick();

    expect(screen.getByLabelText(/Name/i)).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText(/Name/i)).toHaveAttribute(
      "aria-describedby",
      "create-reading-list-error",
    );
  });

  it("hides the form after Cancel", async () => {
    vi.mocked(readingListStore).loaded = true;
    render(ReadingLists);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New List/i }));
    await tick();
    fireEvent.click(screen.getByRole("button", { name: /Cancel/i }));
    await tick();

    expect(
      screen.queryByRole("heading", { name: /Create Reading List/i }),
    ).not.toBeInTheDocument();
  });

  it("renders ReadingListDetail when subPath is a list ID", async () => {
    vi.mocked(readingListStore).loaded = true;
    vi.mocked(routerStore).subPath = "rl-1";
    render(ReadingLists);
    await tick();

    // When subPath is set, the component renders ReadingListDetail (mocked).
    // We just confirm the list grid is NOT rendered.
    expect(screen.queryByText(/No reading lists yet/i)).not.toBeInTheDocument();
  });
});
