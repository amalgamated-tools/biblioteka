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
vi.mock("./libraries/FirstLibraryWizard.svelte", () => ({ default: () => {} }));

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

  it("renders an sr-only h1 heading in empty state", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "";
    render(Libraries);
    await tick();

    expect(
      screen.getByRole("heading", { level: 1, name: /Libraries/i }),
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

describe("Libraries – setup mode", () => {
  const mockLib = {
    id: "lib-1",
    name: "Fiction",
    paths: ["/books"],
    organization_type: "book_per_folder" as const,
    monitored: false,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };

  beforeEach(() => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "setup";
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("does not show the 'Add A Library' button in setup mode", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "setup";
    render(Libraries);
    await tick();

    expect(screen.queryByRole("button", { name: /Add A Library/i })).toBeNull();
  });

  it("redirects to 'libraries' when setup mode is active but libraries already exist", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [mockLib];
    vi.mocked(routerStore).subPath = "setup";
    render(Libraries);
    await tick();

    expect(vi.mocked(routerStore).navigate).toHaveBeenCalledWith("libraries");
  });

  it("does not redirect when setup mode is active and no libraries exist", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(routerStore).subPath = "setup";
    render(Libraries);
    await tick();

    expect(vi.mocked(routerStore).navigate).not.toHaveBeenCalled();
  });

  it("does not redirect when setup mode is active but libraries are not yet loaded", async () => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [mockLib];
    vi.mocked(routerStore).subPath = "setup";
    render(Libraries);
    await tick();

    expect(vi.mocked(routerStore).navigate).not.toHaveBeenCalled();
  });
});
