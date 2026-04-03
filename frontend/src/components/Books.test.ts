import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    queryParams: new URLSearchParams(),
    setQueryParam: vi.fn(),
  },
}));

vi.mock("../stores/libraries.svelte", () => ({
  libraryStore: {
    isScanning: false,
  },
}));

vi.mock("../lib/api", () => ({
  listBooks: vi.fn().mockResolvedValue({
    books: [],
    total: 0,
    limit: 24,
    offset: 0,
  }),
}));

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  LayoutGrid: () => {},
  List: () => {},
  ChevronLeft: () => {},
  ChevronRight: () => {},
}));

import Books from "./Books.svelte";
import { libraryStore } from "../stores/libraries.svelte";
import * as api from "../lib/api";

describe("Books", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the 'All Books' heading", async () => {
    render(Books);
    await tick();

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("All Books");
  });

  it("calls listBooks on mount", async () => {
    render(Books);
    await tick();
    await tick();

    expect(api.listBooks).toHaveBeenCalled();
  });

  it("passes no pollingInterval to BookList when library is not scanning", async () => {
    vi.mocked(libraryStore).isScanning = false;
    render(Books);
    await tick();
    await tick();

    // With no polling, listBooks is called exactly once (initial load only)
    expect(api.listBooks).toHaveBeenCalledTimes(1);
  });
});
