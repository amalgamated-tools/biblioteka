import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
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
  X: () => {},
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

  it("renders the search input", async () => {
    render(Books);
    await tick();

    expect(
      screen.getByRole("searchbox", { name: "Search books" }),
    ).toBeInTheDocument();
  });

  it("initializes search input from URL query param", async () => {
    const { routerStore } = await import("../stores/router.svelte");
    (routerStore.queryParams as unknown as URLSearchParams).set(
      "query",
      "tolkien",
    );

    render(Books);
    await tick();

    expect(screen.getByRole("searchbox", { name: "Search books" })).toHaveValue(
      "tolkien",
    );

    (routerStore.queryParams as unknown as URLSearchParams).delete("query");
  });

  it("shows clear button when search input has a value", async () => {
    render(Books);
    await tick();

    const input = screen.getByRole("searchbox", { name: "Search books" });
    await fireEvent.input(input, { target: { value: "dune" } });
    await tick();

    expect(
      screen.getByRole("button", { name: "Clear search" }),
    ).toBeInTheDocument();
  });

  it("hides clear button when search input is empty", async () => {
    render(Books);
    await tick();

    expect(
      screen.queryByRole("button", { name: "Clear search" }),
    ).not.toBeInTheDocument();
  });
});
