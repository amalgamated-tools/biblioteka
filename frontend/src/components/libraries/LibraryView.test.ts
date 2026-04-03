import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";
import { SvelteSet } from "svelte/reactivity";
import type { Library } from "../../types";

vi.mock("../../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("../../stores/libraries.svelte", () => ({
  libraryStore: {
    scanningIds: { has: vi.fn().mockReturnValue(false) },
    clearScanning: vi.fn(),
  },
}));

vi.mock("../../lib/api", () => ({
  listLibraryBooks: vi.fn().mockResolvedValue({
    books: [],
    total: 0,
    limit: 24,
    offset: 0,
  }),
}));

vi.mock("lucide-svelte", () => ({
  Library: () => {},
  Settings2: () => {},
  BookOpen: () => {},
  LayoutGrid: () => {},
  List: () => {},
  ChevronLeft: () => {},
  ChevronRight: () => {},
}));

import LibraryView from "./LibraryView.svelte";
import { routerStore } from "../../stores/router.svelte";
import { libraryStore } from "../../stores/libraries.svelte";

const fakeLibrary: Library = {
  id: "lib-1",
  name: "Science Fiction",
  paths: ["/books/scifi"],
  organization_type: "book_per_folder",
  monitored: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("LibraryView", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the library name in the heading", async () => {
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: null },
    });
    await tick();

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "Science Fiction",
    );
  });

  it("renders 'Library' as a fallback when library is null", async () => {
    render(LibraryView, {
      props: { library: null, libraryId: "lib-1", error: null },
    });
    await tick();

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Library");
  });

  it("renders the library settings button", async () => {
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: null },
    });
    await tick();

    expect(
      screen.getByRole("button", { name: "Library settings" }),
    ).toBeInTheDocument();
  });

  it("navigates to the edit page when the settings button is clicked", async () => {
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: null },
    });
    await tick();

    await fireEvent.click(screen.getByRole("button", { name: "Library settings" }));
    expect(routerStore.navigate).toHaveBeenCalledWith("libraries/edit/lib-1");
  });

  it("shows an error banner when the error prop is set", async () => {
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: "Failed to load" },
    });
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Failed to load");
  });

  it("does not show an error banner when the error prop is null", async () => {
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: null },
    });
    await tick();

    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("calls listLibraryBooks on mount", async () => {
    const { listLibraryBooks } = await import("../../lib/api");
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: null },
    });
    await tick();
    await tick();

    expect(listLibraryBooks).toHaveBeenCalled();
  });

  it("does not call clearScanning when library is not in scanningIds", async () => {
    vi.mocked(libraryStore).scanningIds = { has: vi.fn().mockReturnValue(false) } as unknown as SvelteSet<string>;
    render(LibraryView, {
      props: { library: fakeLibrary, libraryId: "lib-1", error: null },
    });
    await tick();
    await tick();

    expect(libraryStore.clearScanning).not.toHaveBeenCalled();
  });
});
