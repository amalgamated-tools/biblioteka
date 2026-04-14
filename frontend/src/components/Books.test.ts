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

vi.mock("./ui/BookList.svelte", () => ({ default: () => {} }));

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  Search: () => {},
}));

import Books from "./Books.svelte";

describe("Books", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the 'All Books' heading", async () => {
    render(Books);
    await tick();

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "All Books",
    );
  });

  it("renders a search input for filtering books", async () => {
    render(Books);
    await tick();

    const input = screen.getByRole("searchbox", { name: "Search books" });
    expect(input).toBeInTheDocument();
  });
});
