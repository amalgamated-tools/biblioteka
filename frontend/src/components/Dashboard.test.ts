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
    navigate: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({
  LayoutDashboard: () => {},
  Library: () => {},
  Plus: () => {},
  ArrowRight: () => {},
}));

import Dashboard from "./Dashboard.svelte";
import { libraryStore } from "../stores/libraries.svelte";
import { routerStore } from "../stores/router.svelte";

describe("Dashboard", () => {
  beforeEach(() => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [];
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the Dashboard heading", () => {
    render(Dashboard);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "Dashboard",
    );
  });

  it("shows the onboarding card when libraries are loaded and empty", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    render(Dashboard);
    await tick();

    expect(
      screen.getByRole("heading", { name: /Get started with Biblioteka/i }),
    ).toBeInTheDocument();
  });

  it("shows 'Add Your First Library' button in empty state", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    render(Dashboard);
    await tick();

    expect(
      screen.getByRole("button", { name: /Add Your First Library/i }),
    ).toBeInTheDocument();
  });

  it("navigates to libraries/new when the onboarding button is clicked", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    render(Dashboard);
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: /Add Your First Library/i }),
    );
    expect(routerStore.navigate).toHaveBeenCalledWith("libraries/new");
  });

  it("shows stats grid when libraries exist", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [
      {
        id: "lib-1",
        name: "Fiction",
        paths: ["/books/fiction"],
        organization_type: "book_per_folder",
        monitored: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];
    render(Dashboard);
    await tick();

    expect(screen.getByText("Total Books")).toBeInTheDocument();
    expect(screen.getByText("Libraries")).toBeInTheDocument();
    expect(screen.getByText("Currently Reading")).toBeInTheDocument();
  });

  it("does not show the onboarding card when libraries exist", async () => {
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
    render(Dashboard);
    await tick();

    expect(screen.queryByRole("heading", { name: /Get started/i })).toBeNull();
  });

  it("shows the library count in the stats grid", async () => {
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
      {
        id: "lib-2",
        name: "Non-Fiction",
        paths: [],
        organization_type: "book_per_folder",
        monitored: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];
    render(Dashboard);
    await tick();

    // The Libraries stat card should show the count
    const statValues = screen.getAllByText("2");
    expect(statValues.length).toBeGreaterThan(0);
  });

  it("triggers libraryStore.load() when libraries are not yet loaded", async () => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [];
    render(Dashboard);
    await tick();

    expect(libraryStore.load).toHaveBeenCalled();
  });
});
