import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import {
  cleanup,
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/svelte";
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

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Clear search" }),
      ).toBeInTheDocument();
    });
  });

  it("hides clear button when search input is empty", async () => {
    render(Books);
    await tick();

    expect(
      screen.queryByRole("button", { name: "Clear search" }),
    ).not.toBeInTheDocument();
  });
});

describe("Books debounce and URL sync", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
    vi.clearAllMocks();
  });

  it("calls setQueryParam only after the 300ms debounce", async () => {
    const { routerStore } = await import("../stores/router.svelte");
    render(Books);
    await tick();

    const input = screen.getByRole("searchbox", { name: "Search books" });
    await fireEvent.input(input, { target: { value: "dune" } });
    await tick();
    await tick();

    // Not called yet — still within the debounce window
    expect(routerStore.setQueryParam).not.toHaveBeenCalledWith("query", "dune");

    // Advance past the 300ms debounce
    await vi.advanceTimersByTimeAsync(300);
    await tick();

    expect(routerStore.setQueryParam).toHaveBeenCalledWith("query", "dune");
  });

  it("resets debounce timer on subsequent input within 300ms", async () => {
    const { routerStore } = await import("../stores/router.svelte");
    render(Books);
    await tick();

    const input = screen.getByRole("searchbox", { name: "Search books" });

    // Type "du" then wait 200ms, then type "dune"
    await fireEvent.input(input, { target: { value: "du" } });
    await tick();
    await tick();
    await vi.advanceTimersByTimeAsync(200);

    await fireEvent.input(input, { target: { value: "dune" } });
    await tick();
    await tick();

    // Advance another 200ms — the first timer would have fired at 300ms but
    // was reset, so nothing should have been called with "du"
    await vi.advanceTimersByTimeAsync(200);
    await tick();
    expect(routerStore.setQueryParam).not.toHaveBeenCalledWith("query", "du");

    // Advance the remaining 100ms for the second timer
    await vi.advanceTimersByTimeAsync(100);
    await tick();
    expect(routerStore.setQueryParam).toHaveBeenCalledWith("query", "dune");
  });

  it("clearSearch calls setQueryParam with null and cancels pending debounce", async () => {
    const { routerStore } = await import("../stores/router.svelte");
    render(Books);
    await tick();

    const input = screen.getByRole("searchbox", { name: "Search books" });

    // Type something to start a debounce
    await fireEvent.input(input, { target: { value: "tolkien" } });
    await tick();
    await tick();

    // Click clear before debounce fires
    const clearButton = screen.getByRole("button", { name: "Clear search" });
    await fireEvent.click(clearButton);
    await tick();

    expect(routerStore.setQueryParam).toHaveBeenCalledWith("query", null);

    // Advance past the debounce — the old timer should have been canceled
    await vi.advanceTimersByTimeAsync(300);
    await tick();

    // setQueryParam should NOT have been called with "tolkien"
    expect(routerStore.setQueryParam).not.toHaveBeenCalledWith(
      "query",
      "tolkien",
    );
  });

  it("sets query param to null when input is cleared to empty string", async () => {
    const { routerStore } = await import("../stores/router.svelte");
    render(Books);
    await tick();

    const input = screen.getByRole("searchbox", { name: "Search books" });
    await fireEvent.input(input, { target: { value: "x" } });
    await tick();
    await tick();
    await vi.advanceTimersByTimeAsync(300);
    await tick();

    // Now clear by typing empty
    await fireEvent.input(input, { target: { value: "" } });
    await tick();
    await tick();
    await vi.advanceTimersByTimeAsync(300);
    await tick();

    expect(routerStore.setQueryParam).toHaveBeenCalledWith("query", null);
  });
});
