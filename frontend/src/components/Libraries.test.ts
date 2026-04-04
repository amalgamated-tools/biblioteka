import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../stores/libraries.svelte", () => ({
  libraryStore: {
    loaded: false,
    libraries: [],
    load: vi.fn(),
  },
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "",
    navigate: vi.fn(),
  },
}));

vi.mock("./ui/AlertBanner.svelte", () => ({ default: () => {} }));
vi.mock("./libraries/LibraryView.svelte", () => ({ default: () => {} }));
vi.mock("./libraries/LibraryForm.svelte", () => ({ default: () => {} }));

vi.mock("lucide-svelte", () => ({
  Plus: () => {},
  Library: () => {},
}));

import Libraries from "./Libraries.svelte";
import { libraryStore } from "../stores/libraries.svelte";
import { routerStore } from "../stores/router.svelte";

describe("Libraries", () => {
  beforeEach(() => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "";
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows 'Add A Library' button in empty state with no libraries", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "";
    render(Libraries);
    await tick();

    expect(
      screen.getByRole("button", { name: /Add A Library/i }),
    ).toBeInTheDocument();
  });

  it("navigates to libraries/new when the button is clicked", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "";
    render(Libraries);
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: /Add A Library/i }),
    );
    expect(routerStore.navigate).toHaveBeenCalledWith("libraries/new");
  });

  it("shows informational text when libraries exist and subPath is empty", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [
      {
        id: "lib-1",
        name: "Fiction",
        paths: [],
        organization_type: "book_per_folder",
        monitored: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];
    vi.mocked(routerStore).subPath = "";
    render(Libraries);
    await tick();

    expect(
      screen.getByText(/Select a library from the sidebar/i),
    ).toBeInTheDocument();
  });

  it("calls libraryStore.load() when libraries are not yet loaded", async () => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "";
    render(Libraries);
    await tick();

    expect(libraryStore.load).toHaveBeenCalled();
  });

  it("does not call libraryStore.load() when libraries are already loaded", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "";
    render(Libraries);
    await tick();

    expect(libraryStore.load).not.toHaveBeenCalled();
  });
});
