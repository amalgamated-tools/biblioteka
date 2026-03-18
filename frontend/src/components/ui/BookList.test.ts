import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  LayoutGrid: () => {},
  List: () => {},
  ChevronLeft: () => {},
  ChevronRight: () => {},
}));

vi.mock("./AlertBanner.svelte", () => ({ default: () => {} }));
vi.mock("./BookCard.svelte", () => ({ default: () => {} }));

import BookList from "./BookList.svelte";

describe("BookList accessibility", () => {
  it("exposes loading state through status and busy attributes", () => {
    const neverResolves = vi.fn(
      () =>
        new Promise<never>(() => {
          // Keep loading state active for assertions.
        }),
    );

    const { container } = render(BookList, {
      props: { fetchBooks: neverResolves },
    });

    const loadingContainer = container.querySelector('[aria-busy="true"]');
    expect(loadingContainer).toBeInTheDocument();
    expect(loadingContainer).toHaveAttribute("aria-label", "Loading books");
    expect(screen.getByRole("status")).toHaveTextContent("Loading books...");
  });
});
