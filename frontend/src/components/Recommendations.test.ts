import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";

vi.mock("../lib/api", () => ({
  getRecommendations: vi.fn(),
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({
  Sparkles: () => null,
}));

import Recommendations from "./Recommendations.svelte";
import { getRecommendations } from "../lib/api";
import { routerStore } from "../stores/router.svelte";
import type { BookSummary } from "../types";

const fakeBook: BookSummary = {
  id: "b1",
  title: "Dune",
  description: null,
  asin: null,
  isbn10: null,
  isbn13: null,
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: null,
  publisher: null,
  language: null,
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeBookWithCover: BookSummary = {
  ...fakeBook,
  id: "b2",
  title: "Foundation",
  cover_image_url: "https://example.com/cover.jpg",
};

describe("Recommendations", () => {
  beforeEach(() => {
    vi.mocked(getRecommendations).mockReturnValue(new Promise(() => {}));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the section heading", async () => {
    vi.mocked(getRecommendations).mockResolvedValue([]);
    render(Recommendations);

    expect(screen.getByText("You Might Also Like")).toBeInTheDocument();
  });

  describe("loading state", () => {
    it("renders skeleton list while fetch is pending", () => {
      render(Recommendations);

      const skeleton = screen.getByRole("list", {
        name: "Loading recommendations",
      });
      expect(skeleton).toBeInTheDocument();
      expect(skeleton).toHaveAttribute("aria-busy", "true");
    });

    it("renders 5 skeleton items", () => {
      render(Recommendations);

      const skeleton = screen.getByRole("list", {
        name: "Loading recommendations",
      });
      expect(skeleton.querySelectorAll("li")).toHaveLength(5);
    });

    it("does not show the recommended books list while loading", () => {
      render(Recommendations);

      expect(
        screen.queryByRole("list", { name: "Recommended books" }),
      ).not.toBeInTheDocument();
    });
  });

  describe("error state", () => {
    it("shows the error message when fetch fails", async () => {
      vi.mocked(getRecommendations).mockRejectedValue(
        new Error("Network error"),
      );
      render(Recommendations);

      await waitFor(() => {
        expect(screen.getByText("Network error")).toBeInTheDocument();
      });
    });

    it("shows a fallback message for non-Error rejections", async () => {
      vi.mocked(getRecommendations).mockRejectedValue("oops");
      render(Recommendations);

      await waitFor(() => {
        expect(
          screen.getByText("Failed to load recommendations"),
        ).toBeInTheDocument();
      });
    });

    it("hides the skeleton list after an error", async () => {
      vi.mocked(getRecommendations).mockRejectedValue(
        new Error("Network error"),
      );
      render(Recommendations);

      await waitFor(() => {
        expect(
          screen.queryByRole("list", { name: "Loading recommendations" }),
        ).not.toBeInTheDocument();
      });
    });
  });

  describe("empty state", () => {
    it("shows the empty-state prompt when no recommendations exist", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([]);
      render(Recommendations);

      await waitFor(() => {
        expect(
          screen.getByText(/Read some books to get personalized recommendations/i),
        ).toBeInTheDocument();
      });
    });

    it("renders a link to Settings → KOSync in the empty state", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([]);
      render(Recommendations);

      await waitFor(() => {
        const link = screen.getByRole("link", { name: /Settings.*KOSync/i });
        expect(link).toHaveAttribute("href", "#settings/kobo");
      });
    });

    it("hides the skeleton list in the empty state", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([]);
      render(Recommendations);

      await waitFor(() => {
        expect(
          screen.queryByRole("list", { name: "Loading recommendations" }),
        ).not.toBeInTheDocument();
      });
    });
  });

  describe("books list", () => {
    it("renders a book card for each recommendation", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([fakeBook, fakeBookWithCover]);
      render(Recommendations);

      await waitFor(() => {
        expect(
          screen.getByRole("list", { name: "Recommended books" }),
        ).toBeInTheDocument();
      });

      expect(
        screen.getByRole("button", { name: "View Dune" }),
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "View Foundation" }),
      ).toBeInTheDocument();
    });

    it("renders book titles as visible text", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([fakeBook]);
      render(Recommendations);

      await waitFor(() => {
        expect(screen.getByText("Dune")).toBeInTheDocument();
      });
    });

    it("renders a cover image when cover_image_url is provided", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([fakeBookWithCover]);
      const { container } = render(Recommendations);

      await waitFor(() => {
        const img = container.querySelector("img");
        expect(img).not.toBeNull();
        expect(img).toHaveAttribute("src", fakeBookWithCover.cover_image_url);
      });
    });

    it("calls routerStore.navigate with the correct book path on click", async () => {
      const user = userEvent.setup();
      vi.mocked(getRecommendations).mockResolvedValue([fakeBook]);
      render(Recommendations);

      await waitFor(() => {
        expect(
          screen.getByRole("button", { name: "View Dune" }),
        ).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "View Dune" }));
      await tick();

      expect(vi.mocked(routerStore).navigate).toHaveBeenCalledWith(
        `books/${fakeBook.id}`,
      );
    });

    it("calls getRecommendations with the default limit of 10", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([]);
      render(Recommendations);
      await tick();

      expect(vi.mocked(getRecommendations)).toHaveBeenCalledWith(10);
    });

    it("calls getRecommendations with a custom limit when provided", async () => {
      vi.mocked(getRecommendations).mockResolvedValue([]);
      render(Recommendations, { props: { limit: 5 } });
      await tick();

      expect(vi.mocked(getRecommendations)).toHaveBeenCalledWith(5);
    });
  });
});
