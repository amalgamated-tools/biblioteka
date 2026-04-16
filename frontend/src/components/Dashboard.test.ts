import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import {
  cleanup,
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/svelte";
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

vi.mock("../stores/auth.svelte", () => ({
  authStore: {
    user: {
      id: "user-1",
      name: "Test User",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    },
  },
}));

vi.mock("../stores/onboarding.svelte", () => ({
  onboardingStore: {
    isSkipped: vi.fn().mockReturnValue(false),
  },
}));

vi.mock("../lib/api", () => ({
  getTotalBooksCount: vi.fn().mockResolvedValue(0),
  getDownloadsPerMonth: vi.fn().mockResolvedValue([]),
  getReadingProgressStats: vi.fn().mockResolvedValue({
    current_streak: 0,
    total_tracked: 0,
    total_finished: 0,
    in_progress: [],
  }),
  getYearInBooks: vi.fn().mockResolvedValue({
    year: 2026,
    books_finished: 0,
    active_days: 0,
    longest_streak: 0,
    total_downloads: 0,
  }),
  getRecommendations: vi.fn().mockResolvedValue([]),
}));

vi.mock("lucide-svelte", () => ({
  LayoutDashboard: () => {},
  Library: () => {},
  Plus: () => {},
  ArrowRight: () => {},
  Flame: () => {},
  BookOpen: () => {},
  CheckCheck: () => {},
  CalendarDays: () => {},
  Download: () => {},
  Sparkles: () => {},
}));

import Dashboard from "./Dashboard.svelte";
import { libraryStore } from "../stores/libraries.svelte";
import { routerStore } from "../stores/router.svelte";
import { onboardingStore } from "../stores/onboarding.svelte";
import {
  getTotalBooksCount,
  getDownloadsPerMonth,
  getReadingProgressStats,
  getYearInBooks,
  getRecommendations,
} from "../lib/api";

describe("Dashboard", () => {
  beforeEach(() => {
    vi.mocked(libraryStore).loaded = false;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(onboardingStore).isSkipped = vi.fn().mockReturnValue(false);
    vi.mocked(getTotalBooksCount).mockResolvedValue(0);
    vi.mocked(getDownloadsPerMonth).mockResolvedValue([]);
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 0,
      total_tracked: 0,
      total_finished: 0,
      in_progress: [],
    });
    vi.mocked(getYearInBooks).mockResolvedValue({
      year: 2026,
      books_finished: 0,
      active_days: 0,
      longest_streak: 0,
      total_downloads: 0,
    });
    vi.mocked(getRecommendations).mockResolvedValue([]);
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

  it("navigates to libraries/setup when the onboarding button is clicked", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    render(Dashboard);
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: /Add Your First Library/i }),
    );
    expect(routerStore.navigate).toHaveBeenCalledWith("libraries/setup");
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
    expect(screen.queryByText("Currently Reading")).toBeNull();
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

  it("uses semantic dl/dt/dd structure for stat cards", async () => {
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

    const terms = screen.getAllByRole("term");
    const definitions = screen.getAllByRole("definition");

    expect(terms).toHaveLength(2);
    expect(definitions).toHaveLength(2);

    const termTexts = terms.map((el) => el.textContent?.trim());
    expect(termTexts).toContain("Total Books");
    expect(termTexts).toContain("Libraries");
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

  it("shows loading placeholder while total books is being fetched", async () => {
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
    // Never resolves, so totalBooks stays null → "…"
    vi.mocked(getTotalBooksCount).mockReturnValue(
      new Promise<number>(() => {}),
    );
    render(Dashboard);
    await tick();

    expect(screen.getByText("…")).toBeInTheDocument();
  });

  it("shows the real total books count after fetching", async () => {
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
    vi.mocked(getTotalBooksCount).mockResolvedValue(500);
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText("500")).toBeInTheDocument();
    });

    expect(getTotalBooksCount).toHaveBeenCalled();
  });

  it("shows the downloads histogram when data is available", async () => {
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
    vi.mocked(getDownloadsPerMonth).mockResolvedValue([
      { month: "2026-02", count: 3 },
      { month: "2026-03", count: 5 },
    ]);
    render(Dashboard);

    await waitFor(() => {
      expect(
        screen.getByTestId("downloads-histogram-card"),
      ).toBeInTheDocument();
    });
  });

  it("shows a downloads error banner when the stats fetch fails", async () => {
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
    vi.mocked(getDownloadsPerMonth).mockRejectedValue(
      new Error("network error"),
    );
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText("network error")).toBeInTheDocument();
    });
  });

  // ---- Reading Activity section ----

  const libWithOne = [
    {
      id: "lib-1",
      name: "Fiction",
      paths: [],
      organization_type: "book_per_folder" as const,
      monitored: false,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ];

  it("shows Reading Activity heading when stats load", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 0,
      total_tracked: 0,
      total_finished: 0,
      in_progress: [],
    });
    render(Dashboard);

    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: /Reading Activity/i }),
      ).toBeInTheDocument();
    });
  });

  it("shows KOSync nudge when total_tracked is 0", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 0,
      total_tracked: 0,
      total_finished: 0,
      in_progress: [],
    });
    render(Dashboard);

    await waitFor(() => {
      expect(
        screen.getByText(/No reading activity recorded yet/i),
      ).toBeInTheDocument();
    });
  });

  it("shows streak badge when current_streak > 0", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 5,
      total_tracked: 3,
      total_finished: 0,
      in_progress: [],
    });
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText(/5-day streak/i)).toBeInTheDocument();
    });
  });

  it("shows finished books badge when total_finished > 0", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 0,
      total_tracked: 2,
      total_finished: 2,
      in_progress: [],
    });
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText(/2 books finished/i)).toBeInTheDocument();
    });
  });

  it("uses accessible contrast classes for tracked documents text", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 0,
      total_tracked: 2,
      total_finished: 0,
      in_progress: [],
    });
    render(Dashboard);

    await waitFor(() => {
      const tracked = screen.getByText(/documents tracked/i);
      expect(tracked).toHaveClass("text-ink-500");
      expect(tracked).toHaveClass("dark:text-ink-300");
    });
  });

  it("shows Currently Reading list with document names", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 1,
      total_tracked: 1,
      total_finished: 0,
      in_progress: [
        {
          document: "my-great-book",
          percentage: 0.42,
          device: "KOReader",
          last_synced: "2026-04-12T10:00:00Z",
          estimated_minutes_remaining: null,
        },
      ],
    });
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText("my-great-book")).toBeInTheDocument();
    });
    expect(screen.getByText("42%")).toBeInTheDocument();
    expect(screen.getByText("KOReader")).toBeInTheDocument();
    const helperTextRow = screen.getByText("KOReader").closest("div");
    expect(helperTextRow).toHaveClass("text-ink-500");
    expect(helperTextRow).toHaveClass("dark:text-ink-400");
  });

  it("shows estimated time remaining when provided", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockResolvedValue({
      current_streak: 1,
      total_tracked: 1,
      total_finished: 0,
      in_progress: [
        {
          document: "timed-book",
          percentage: 0.5,
          device: null,
          last_synced: "2026-04-12T10:00:00Z",
          estimated_minutes_remaining: 45,
        },
      ],
    });
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText("~45m left")).toBeInTheDocument();
    });
  });

  it("does not show reading activity section while stats are still loading", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    // Never resolves — simulates loading state.
    vi.mocked(getReadingProgressStats).mockReturnValue(new Promise(() => {}));
    render(Dashboard);
    await tick();

    expect(
      screen.queryByRole("heading", { name: /Reading Activity/i }),
    ).toBeNull();
    // Welcome fallback should be shown instead.
    expect(
      screen.getByRole("heading", { name: /Welcome to Biblioteka/i }),
    ).toBeInTheDocument();
  });

  it("shows a reading stats error banner when the fetch fails", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getReadingProgressStats).mockRejectedValue(
      new Error("reading stats unavailable"),
    );
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText("reading stats unavailable")).toBeInTheDocument();
    });
    // The "Welcome to Biblioteka" fallback should NOT appear alongside the error.
    expect(
      screen.queryByRole("heading", { name: /Welcome to Biblioteka/i }),
    ).toBeNull();
  });

  it("shows a year-in-books error banner when the fetch fails", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = libWithOne;
    vi.mocked(getYearInBooks).mockRejectedValue(
      new Error("year in books unavailable"),
    );
    render(Dashboard);

    await waitFor(() => {
      expect(screen.getByText("year in books unavailable")).toBeInTheDocument();
    });
  });

  describe("Year in Books", () => {
    it("does not show year-in-books card when all stats are zero", async () => {
      vi.mocked(libraryStore).loaded = true;
      vi.mocked(libraryStore).libraries = libWithOne;
      vi.mocked(getYearInBooks).mockResolvedValue({
        year: 2026,
        books_finished: 0,
        active_days: 0,
        longest_streak: 0,
        total_downloads: 0,
      });
      render(Dashboard);
      await tick();

      await waitFor(() => {
        expect(
          screen.queryByTestId("year-in-books-card"),
        ).not.toBeInTheDocument();
      });
    });

    it("shows year-in-books card when there are books finished", async () => {
      vi.mocked(libraryStore).loaded = true;
      vi.mocked(libraryStore).libraries = libWithOne;
      vi.mocked(getYearInBooks).mockResolvedValue({
        year: 2026,
        books_finished: 3,
        active_days: 20,
        longest_streak: 5,
        total_downloads: 8,
      });
      render(Dashboard);

      await waitFor(() => {
        expect(screen.getByTestId("year-in-books-card")).toBeInTheDocument();
      });
      expect(screen.getByText("2026 in Books")).toBeInTheDocument();
      expect(screen.getByText("3")).toBeInTheDocument();
      expect(screen.getByText("books finished")).toBeInTheDocument();
      expect(screen.getByText("20")).toBeInTheDocument();
      expect(screen.getByText("days reading")).toBeInTheDocument();
      expect(screen.getByText("5")).toBeInTheDocument();
      expect(screen.getByText("days longest streak")).toBeInTheDocument();
      expect(screen.getByText("8")).toBeInTheDocument();
      expect(screen.getByText("downloads")).toBeInTheDocument();
      const subheading = screen.getByText("Your reading year at a glance");
      expect(subheading).toHaveClass("text-ink-500");
      expect(subheading).toHaveClass("dark:text-ink-400");
    });

    it("shows year-in-books card when there are downloads only", async () => {
      vi.mocked(libraryStore).loaded = true;
      vi.mocked(libraryStore).libraries = libWithOne;
      vi.mocked(getYearInBooks).mockResolvedValue({
        year: 2026,
        books_finished: 0,
        active_days: 0,
        longest_streak: 0,
        total_downloads: 5,
      });
      render(Dashboard);

      await waitFor(() => {
        expect(screen.getByTestId("year-in-books-card")).toBeInTheDocument();
      });
    });

    it("uses singular 'book' when exactly one book is finished", async () => {
      vi.mocked(libraryStore).loaded = true;
      vi.mocked(libraryStore).libraries = libWithOne;
      vi.mocked(getYearInBooks).mockResolvedValue({
        year: 2026,
        books_finished: 1,
        active_days: 10,
        longest_streak: 3,
        total_downloads: 1,
      });
      render(Dashboard);

      await waitFor(() => {
        expect(screen.getByText("book finished")).toBeInTheDocument();
        expect(screen.getByText("download")).toBeInTheDocument();
      });
    });
  });

  // ---- Onboarding / skip-state ----

  it("shows 'Add Your First Library' button when libraries are empty and not skipped", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(onboardingStore).isSkipped = vi.fn().mockReturnValue(false);
    render(Dashboard);
    await tick();

    expect(
      screen.getByRole("button", { name: /Add Your First Library/i }),
    ).toBeInTheDocument();
  });

  it("shows 'No libraries yet' heading when libraries are empty and skipped", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(onboardingStore).isSkipped = vi.fn().mockReturnValue(true);
    render(Dashboard);
    await tick();

    expect(
      screen.getByRole("heading", { name: /No libraries yet/i }),
    ).toBeInTheDocument();
  });

  it("shows 'Set up your first library' button in skipped state", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(onboardingStore).isSkipped = vi.fn().mockReturnValue(true);
    render(Dashboard);
    await tick();

    expect(
      screen.getByRole("button", { name: /Set up your first library/i }),
    ).toBeInTheDocument();
  });

  it("navigates to libraries/setup from the skipped-state button", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(onboardingStore).isSkipped = vi.fn().mockReturnValue(true);
    render(Dashboard);
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: /Set up your first library/i }),
    );
    expect(routerStore.navigate).toHaveBeenCalledWith("libraries/setup");
  });

  it("does not show the 'Get started' wizard card in skipped state", async () => {
    vi.mocked(libraryStore).loaded = true;
    vi.mocked(libraryStore).libraries = [];
    vi.mocked(onboardingStore).isSkipped = vi.fn().mockReturnValue(true);
    render(Dashboard);
    await tick();

    expect(
      screen.queryByRole("heading", { name: /Get started with Biblioteka/i }),
    ).toBeNull();
  });
});
