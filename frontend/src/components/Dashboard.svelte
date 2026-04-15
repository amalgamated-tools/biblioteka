<script lang="ts">
  import {
    LayoutDashboard,
    Library,
    Plus,
    ArrowRight,
    Flame,
    BookOpen,
    CheckCheck,
    CalendarDays,
    Download,
  } from "lucide-svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import { routerStore } from "../stores/router.svelte";
  import {
    getTotalBooksCount,
    getDownloadsPerMonth,
    getReadingProgressStats,
    getYearInBooks,
  } from "../lib/api";
  import type {
    MonthlyDownloads,
    ReadingProgressStats,
    YearInBooks,
  } from "../types";
  import AlertBanner from "./ui/AlertBanner.svelte";
  import DownloadsHistogram from "./ui/DownloadsHistogram.svelte";

  let totalBooks = $state<number | null>(null);
  let countError: string | null = $state(null);
  let monthlyDownloads = $state<MonthlyDownloads[]>([]);
  let downloadsError: string | null = $state(null);
  let readingStats = $state<ReadingProgressStats | null>(null);
  let yearInBooks = $state<YearInBooks | null>(null);

  $effect(() => {
    if (!libraryStore.loaded) {
      libraryStore.load();
    }
  });

  let countFetched = false;
  $effect(() => {
    if (!countFetched) {
      countFetched = true;
      getTotalBooksCount()
        .then((count) => {
          totalBooks = count;
        })
        .catch((err) => {
          console.error("Failed to fetch total books count:", err);
          countError =
            err instanceof Error ? err.message : "Failed to load book count";
          totalBooks = 0;
        });
    }
  });

  let downloadsFetched = false;
  $effect(() => {
    if (!downloadsFetched) {
      downloadsFetched = true;
      getDownloadsPerMonth(12)
        .then((data) => {
          monthlyDownloads = data;
        })
        .catch((err) => {
          console.error("Failed to fetch download stats:", err);
          downloadsError =
            err instanceof Error
              ? err.message
              : "Failed to load download stats";
        });
    }
  });

  let statsFetched = false;
  $effect(() => {
    if (!statsFetched) {
      statsFetched = true;
      getReadingProgressStats()
        .then((stats) => {
          readingStats = stats;
        })
        .catch((err) => {
          console.error("Failed to fetch reading stats:", err);
          // Non-fatal: the reading activity section will simply not appear.
        });
    }
  });

  $effect(() => {
    let cancelled = false;
    getYearInBooks()
      .then((data) => {
        if (!cancelled) yearInBooks = data;
      })
      .catch((err) => {
        console.error("Failed to fetch year-in-books stats:", err);
        // Non-fatal: the section will simply not appear.
      });
    return () => {
      cancelled = true;
    };
  });

  const stats = $derived([
    {
      label: "Total Books",
      value: totalBooks === null ? "…" : totalBooks,
    },
    { label: "Libraries", value: libraryStore.libraries.length },
  ]);

  function formatPercent(value: number): string {
    return `${Math.round(value * 100)}%`;
  }

  function formatEstimate(minutes: number | null | undefined): string {
    if (minutes == null) return "";
    if (minutes < 60) return `~${minutes}m left`;
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    return m === 0 ? `~${h}h left` : `~${h}h ${m}m left`;
  }
</script>

<div>
  <div class="flex items-center gap-3 mb-8">
    <div
      class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
    >
      <LayoutDashboard
        class="w-5 h-5 text-accent-600 dark:text-accent-400"
        aria-hidden="true"
      />
    </div>
    <h1
      class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
    >
      Dashboard
    </h1>
  </div>

  {#if libraryStore.loaded && libraryStore.libraries.length === 0}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in-up"
    >
      <div class="flex items-start gap-5">
        <div
          class="w-14 h-14 bg-gradient-to-br from-accent-100 to-accent-200 dark:from-accent-800/30 dark:to-accent-700/20 rounded-2xl flex items-center justify-center flex-shrink-0"
        >
          <Library
            class="w-7 h-7 text-accent-600 dark:text-accent-400"
            aria-hidden="true"
          />
        </div>
        <div>
          <h2
            class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2"
          >
            Get started with Biblioteka
          </h2>
          <p class="text-ink-500 dark:text-ink-300 mb-5 leading-relaxed">
            To begin managing your books, add a library by pointing it to one or
            more folders on your system. Biblioteka will organize the books it
            finds using the Book Per Folder layout.
          </p>
          <button
            onclick={() => routerStore.navigate("libraries/new")}
            class="inline-flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all text-sm font-semibold shadow-md shadow-accent-600/20 hover:shadow-lg hover:shadow-accent-600/30 active:scale-[0.98]"
          >
            <Plus class="w-4 h-4" aria-hidden="true" />
            Add Your First Library
            <ArrowRight class="w-4 h-4" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  {:else}
    {#if countError}
      <AlertBanner variant="error" class="mb-5">{countError}</AlertBanner>
    {/if}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-5 stagger">
      {#each stats as { label, value } (label)}
        <div
          class="group bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 hover:shadow-md hover:border-accent-200 dark:hover:border-accent-800/30 transition-all"
        >
          <dl class="flex flex-col gap-2">
            <dt class="text-sm font-medium text-ink-500 dark:text-ink-300">
              {label}
            </dt>
            <dd
              class="text-4xl font-display font-bold text-ink-900 dark:text-cream-100 m-0"
            >
              {value}
            </dd>
          </dl>
        </div>
      {/each}
    </div>

    {#if downloadsError}
      <AlertBanner variant="error" class="mt-5">{downloadsError}</AlertBanner>
    {:else if monthlyDownloads.length > 0}
      <div
        class="mt-8 bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
        data-testid="downloads-histogram-card"
      >
        <DownloadsHistogram data={monthlyDownloads} />
      </div>
    {/if}

    <!-- Reading Activity section -->
    {#if readingStats !== null}
      <div
        class="mt-8 bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
      >
        <h2
          class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4"
        >
          Reading Activity
        </h2>

        {#if readingStats.total_tracked === 0}
          <!-- Nudge: no reading data recorded yet -->
          <p class="text-ink-500 dark:text-ink-300 leading-relaxed">
            No reading activity recorded yet. Connect KOReader via
            <a
              href="#settings/kobo"
              class="text-accent-600 dark:text-accent-400 underline underline-offset-2 hover:text-accent-700 dark:hover:text-accent-300 transition-colors"
            >
              Settings → KOSync
            </a>
            to start tracking your reading progress.
          </p>
        {:else}
          <!-- Streak + summary badges -->
          <div class="flex flex-wrap items-center gap-4 mb-6">
            {#if readingStats.current_streak > 0}
              <div
                class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-orange-50 dark:bg-orange-900/20 text-orange-600 dark:text-orange-400 rounded-full text-sm font-semibold"
                aria-label="{readingStats.current_streak}-day reading streak"
              >
                <Flame class="w-4 h-4" aria-hidden="true" />
                {readingStats.current_streak}-day streak
              </div>
            {/if}

            {#if readingStats.total_finished > 0}
              <div
                class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 rounded-full text-sm font-semibold"
                aria-label="{readingStats.total_finished} books finished"
              >
                <CheckCheck class="w-4 h-4" aria-hidden="true" />
                {readingStats.total_finished}
                {readingStats.total_finished === 1 ? "book" : "books"} finished
              </div>
            {/if}

            <span class="text-sm text-ink-400 dark:text-ink-500">
              {readingStats.total_tracked}
              {readingStats.total_tracked === 1 ? "document" : "documents"} tracked
            </span>
          </div>

          <!-- Currently-reading list -->
          {#if readingStats.in_progress.length > 0}
            <h3
              class="text-sm font-semibold text-ink-600 dark:text-ink-300 uppercase tracking-wide mb-3"
            >
              Currently Reading
            </h3>
            <ul class="space-y-4" aria-label="Currently reading">
              {#each readingStats.in_progress as item (item.document)}
                <li class="flex flex-col gap-1.5">
                  <div class="flex items-start justify-between gap-3">
                    <div class="flex items-center gap-2 min-w-0">
                      <BookOpen
                        class="w-4 h-4 flex-shrink-0 text-ink-400 dark:text-ink-500"
                        aria-hidden="true"
                      />
                      <span
                        class="text-sm font-medium text-ink-800 dark:text-cream-200 truncate"
                        title={item.document}
                      >
                        {item.document}
                      </span>
                    </div>
                    <span
                      class="text-sm font-semibold text-accent-600 dark:text-accent-400 flex-shrink-0"
                    >
                      {formatPercent(item.percentage)}
                    </span>
                  </div>

                  <!-- Progress bar -->
                  <div
                    class="h-1.5 bg-ink-100 dark:bg-ink-700 rounded-full overflow-hidden"
                    role="progressbar"
                    aria-valuenow={Math.round(item.percentage * 100)}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-label="Reading progress for {item.document}"
                  >
                    <div
                      class="h-full bg-accent-500 rounded-full transition-all"
                      style:width="{item.percentage * 100}%"
                    ></div>
                  </div>

                  <div
                    class="flex items-center gap-3 text-xs text-ink-400 dark:text-ink-500"
                  >
                    {#if item.device}
                      <span>{item.device}</span>
                    {/if}
                    {#if item.estimated_minutes_remaining != null}
                      <span
                        >{formatEstimate(
                          item.estimated_minutes_remaining,
                        )}</span
                      >
                    {/if}
                  </div>
                </li>
              {/each}
            </ul>
          {/if}
        {/if}
      </div>
    {:else}
      <!-- Fallback while reading stats are still loading or unavailable -->
      <div
        class="mt-8 bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
      >
        <h2
          class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-3"
        >
          Welcome to Biblioteka
        </h2>
        <p class="text-ink-500 dark:text-ink-300 leading-relaxed">
          Your personal book management dashboard. Start by adding books to your
          library.
        </p>
      </div>
    {/if}

    <!-- Year in Books section -->
    {#if yearInBooks !== null && (yearInBooks.books_finished > 0 || yearInBooks.active_days > 0 || yearInBooks.total_downloads > 0)}
      <div
        class="mt-8 bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
        data-testid="year-in-books-card"
      >
        <h2
          class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-1"
        >
          {yearInBooks.year} in Books
        </h2>
        <p class="text-sm text-ink-400 dark:text-ink-500 mb-5">
          Your reading year at a glance
        </p>

        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div
            class="flex flex-col items-center gap-1.5 p-4 bg-accent-50 dark:bg-accent-900/20 rounded-xl"
          >
            <CheckCheck
              class="w-5 h-5 text-accent-600 dark:text-accent-400"
              aria-hidden="true"
            />
            <span
              class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
            >
              {yearInBooks.books_finished}
            </span>
            <span class="text-xs text-ink-500 dark:text-ink-400 text-center">
              {yearInBooks.books_finished === 1 ? "book" : "books"} finished
            </span>
          </div>

          <div
            class="flex flex-col items-center gap-1.5 p-4 bg-orange-50 dark:bg-orange-900/20 rounded-xl"
          >
            <Flame
              class="w-5 h-5 text-orange-500 dark:text-orange-400"
              aria-hidden="true"
            />
            <span
              class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
            >
              {yearInBooks.longest_streak}
            </span>
            <span class="text-xs text-ink-500 dark:text-ink-400 text-center">
              {yearInBooks.longest_streak === 1 ? "day" : "days"} longest streak
            </span>
          </div>

          <div
            class="flex flex-col items-center gap-1.5 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl"
          >
            <CalendarDays
              class="w-5 h-5 text-blue-500 dark:text-blue-400"
              aria-hidden="true"
            />
            <span
              class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
            >
              {yearInBooks.active_days}
            </span>
            <span class="text-xs text-ink-500 dark:text-ink-400 text-center">
              days reading
            </span>
          </div>

          <div
            class="flex flex-col items-center gap-1.5 p-4 bg-green-50 dark:bg-green-900/20 rounded-xl"
          >
            <Download
              class="w-5 h-5 text-green-500 dark:text-green-400"
              aria-hidden="true"
            />
            <span
              class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
            >
              {yearInBooks.total_downloads}
            </span>
            <span class="text-xs text-ink-500 dark:text-ink-400 text-center">
              {yearInBooks.total_downloads === 1 ? "download" : "downloads"}
            </span>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>
