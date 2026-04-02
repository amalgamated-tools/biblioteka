<script lang="ts">
  import { untrack } from "svelte";
  import type { BookSummary, PaginatedBooks } from "../../types";
  import {
    BookOpen,
    LayoutGrid,
    List,
    ChevronLeft,
    ChevronRight,
  } from "lucide-svelte";
  import AlertBanner from "./AlertBanner.svelte";
  import BookCard from "./BookCard.svelte";

  interface Props {
    fetchBooks: (limit: number, offset: number) => Promise<PaginatedBooks>;
    pageSize?: number;
    initialOffset?: number;
    onPageChange?: (offset: number) => void;
    /** When set, re-fetches at this interval (ms) while no books are found. */
    pollingInterval?: number;
    /** Called the first time books are found (useful for clearing scanning state). */
    onBooksFound?: () => void;
  }

  const MAX_PAGE_SIZE = 200;

  let {
    fetchBooks,
    pageSize = 24,
    initialOffset = 0,
    onPageChange,
    pollingInterval,
    onBooksFound,
  }: Props = $props();

  let effectivePageSize = $derived(
    Math.max(1, Math.min(pageSize, MAX_PAGE_SIZE)),
  );
  let books: BookSummary[] = $state([]);
  let total = $state(0);
  let offset = $state(
    untrack(() => {
      const raw = Number.isFinite(initialOffset) ? initialOffset : 0;
      const nonNegative = Math.max(0, raw);
      const size = Math.max(1, Math.min(pageSize, MAX_PAGE_SIZE));
      return Math.floor(nonNegative / size) * size;
    }),
  );
  let loading = $state(true);
  let error: string | null = $state(null);
  let viewMode: "grid" | "table" = $state("grid");
  let currentRequestId = 0;
  // Set to true the first time books are found so onBooksFound fires only once.
  let didNotifyBooksFound = false;

  let totalPages = $derived(Math.max(1, Math.ceil(total / effectivePageSize)));
  let currentPage = $derived(Math.floor(offset / effectivePageSize) + 1);
  let rangeStart = $derived(total === 0 ? 0 : offset + 1);
  let rangeEnd = $derived(Math.min(offset + effectivePageSize, total));

  async function load(fetchFn: typeof fetchBooks, size: number, off: number) {
    const requestId = ++currentRequestId;
    loading = true;
    error = null;
    try {
      const data = await fetchFn(size, off);
      if (requestId !== currentRequestId) {
        return;
      }
      books = data.books;
      total = data.total;
      if (data.total > 0 && !didNotifyBooksFound) {
        didNotifyBooksFound = true;
        onBooksFound?.();
      }
      // If offset is past the end (e.g., items deleted), clamp to the last valid page.
      if (data.books.length === 0 && data.total > 0 && off > 0) {
        offset = Math.floor(Math.max(0, data.total - 1) / size) * size;
        return;
      }
    } catch (e) {
      if (requestId !== currentRequestId) {
        return;
      }
      error = e instanceof Error ? e.message : "Failed to load books";
    } finally {
      if (requestId === currentRequestId) {
        loading = false;
      }
    }
  }

  // Track whether this is the first run of the reset effect so that
  // `initialOffset` is honoured on mount.
  let hasInitialized = false;

  // Reset offset when the data source or page size changes (but not on first
  // mount, so that `initialOffset` is respected for the initial page load).
  $effect(() => {
    void fetchBooks;
    void effectivePageSize;
    if (!hasInitialized) {
      hasInitialized = true;
      return;
    }
    offset = 0;
  });

  // Load books whenever offset, page size, or fetch fn changes
  $effect(() => {
    load(fetchBooks, effectivePageSize, offset);
  });

  // Notify the parent whenever the offset changes so it can persist it in the
  // URL. Skips the first run to avoid redundant updates when the component
  // mounts with the same offset already in the URL.
  let hasNotified = false;
  $effect(() => {
    void offset;
    if (!hasNotified) {
      hasNotified = true;
      return;
    }
    onPageChange?.(offset);
  });

  // Poll for books while no books are found and pollingInterval is set.
  // Stops automatically once books appear (total > 0).
  $effect(() => {
    if (!pollingInterval || total > 0) return;

    // Reset so onBooksFound can fire again for this scan cycle.
    didNotifyBooksFound = false;

    let cancelled = false;
    let timerId: ReturnType<typeof setTimeout> | undefined;

    const poll = async () => {
      await load(fetchBooks, effectivePageSize, offset);
      if (cancelled) return;
      timerId = setTimeout(() => {
        void poll();
      }, pollingInterval);
    };

    timerId = setTimeout(() => {
      void poll();
    }, pollingInterval);

    return () => {
      cancelled = true;
      if (timerId !== undefined) {
        clearTimeout(timerId);
      }
    };
  });

  function prevPage() {
    if (currentPage > 1) {
      offset = (currentPage - 2) * effectivePageSize;
    }
  }

  function nextPage() {
    if (currentPage < totalPages) {
      offset = currentPage * effectivePageSize;
    }
  }
</script>

{#if error}
  <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
{:else if loading}
  <div
    class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800"
  >
    <div class="text-center py-8">
      <p class="text-ink-400 dark:text-ink-400">Loading books...</p>
    </div>
  </div>
{:else if total === 0}
  <div
    class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800"
  >
    <div class="text-center py-8">
      {#if pollingInterval}
        <div
          class="w-12 h-12 mx-auto mb-4 rounded-full border-4 border-ink-200 dark:border-ink-700 border-t-accent-500 animate-spin"
          aria-hidden="true"
        ></div>
        <p
          aria-live="polite"
          aria-atomic="true"
          class="text-ink-400 dark:text-ink-400 text-lg"
        >
          Scanning library...
        </p>
        <p class="text-ink-300 dark:text-ink-500 text-sm mt-1">
          Books will appear here once the scan completes.
        </p>
      {:else}
        <BookOpen
          class="w-12 h-12 text-ink-200 dark:text-ink-700 mx-auto mb-4"
        />
        <p class="text-ink-400 dark:text-ink-400 text-lg">No books yet.</p>
        <p class="text-ink-300 dark:text-ink-500 text-sm mt-1">
          Books will appear here once they are added to your libraries.
        </p>
      {/if}
    </div>
  </div>
{:else}
  <!-- Toolbar -->
  <div class="flex items-center justify-between mb-4">
    <p class="text-sm text-ink-400 dark:text-ink-500">
      Showing {rangeStart}–{rangeEnd} of {total} books
    </p>
    <div class="flex items-center gap-1">
      <button
        onclick={() => (viewMode = "grid")}
        class="p-2 rounded-lg transition-colors {viewMode === 'grid'
          ? 'bg-accent-100 dark:bg-accent-800/20 text-accent-600 dark:text-accent-400'
          : 'text-ink-400 hover:text-ink-600 dark:hover:text-ink-200'}"
        title="Grid view"
        aria-label="Grid view"
        aria-pressed={viewMode === "grid"}
      >
        <LayoutGrid class="w-4 h-4" />
      </button>
      <button
        onclick={() => (viewMode = "table")}
        class="p-2 rounded-lg transition-colors {viewMode === 'table'
          ? 'bg-accent-100 dark:bg-accent-800/20 text-accent-600 dark:text-accent-400'
          : 'text-ink-400 hover:text-ink-600 dark:hover:text-ink-200'}"
        title="Table view"
        aria-label="Table view"
        aria-pressed={viewMode === "table"}
      >
        <List class="w-4 h-4" />
      </button>
    </div>
  </div>

  <!-- Grid View -->
  {#if viewMode === "grid"}
    <div
      class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4"
    >
      {#each books as book (book.id)}
        <BookCard {book} />
      {/each}
    </div>
  {:else}
    <!-- Table View -->
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden"
    >
      <table class="w-full text-sm">
        <thead>
          <tr
            class="border-b border-ink-100 dark:border-ink-800 text-left text-ink-400 dark:text-ink-500"
          >
            <th scope="col" class="px-4 py-3 font-medium">Title</th>
            <th scope="col" class="px-4 py-3 font-medium hidden sm:table-cell"
              >Publisher</th
            >
            <th scope="col" class="px-4 py-3 font-medium hidden md:table-cell"
              >Language</th
            >
            <th scope="col" class="px-4 py-3 font-medium hidden lg:table-cell"
              >Published</th
            >
          </tr>
        </thead>
        <tbody>
          {#each books as book (book.id)}
            <tr
              class="border-b border-ink-50 dark:border-ink-800/50 hover:bg-ink-50 dark:hover:bg-ink-800/30 transition-colors"
            >
              <td class="px-4 py-3">
                <div class="flex items-center gap-3">
                  {#if book.cover_image_url}
                    <img
                      src={book.cover_image_url}
                      alt=""
                      class="w-8 h-12 object-cover rounded"
                      loading="lazy"
                    />
                  {:else}
                    <div
                      class="w-8 h-12 bg-ink-100 dark:bg-ink-800 rounded flex items-center justify-center"
                    >
                      <BookOpen
                        class="w-4 h-4 text-ink-300 dark:text-ink-600"
                      />
                    </div>
                  {/if}
                  <span
                    class="font-medium text-ink-900 dark:text-cream-100 truncate max-w-xs"
                    title={book.title}
                  >
                    {book.title}
                  </span>
                </div>
              </td>
              <td
                class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden sm:table-cell truncate max-w-[200px]"
              >
                {book.publisher ?? "—"}
              </td>
              <td
                class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden md:table-cell"
              >
                {book.language ?? "—"}
              </td>
              <td
                class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden lg:table-cell"
              >
                {book.publication_date ?? "—"}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Pagination -->
  {#if totalPages > 1}
    <div class="flex items-center justify-center gap-4 mt-6">
      <button
        onclick={prevPage}
        disabled={currentPage <= 1}
        class="flex items-center gap-1 px-3 py-2 text-sm rounded-lg border border-ink-200 dark:border-ink-700 transition-colors
          {currentPage <= 1
          ? 'text-ink-300 dark:text-ink-600 cursor-not-allowed'
          : 'text-ink-600 dark:text-ink-300 hover:bg-ink-50 dark:hover:bg-ink-800'}"
      >
        <ChevronLeft class="w-4 h-4" />
        Previous
      </button>
      <span class="text-sm text-ink-500 dark:text-ink-400">
        Page {currentPage} of {totalPages}
      </span>
      <button
        onclick={nextPage}
        disabled={currentPage >= totalPages}
        class="flex items-center gap-1 px-3 py-2 text-sm rounded-lg border border-ink-200 dark:border-ink-700 transition-colors
          {currentPage >= totalPages
          ? 'text-ink-300 dark:text-ink-600 cursor-not-allowed'
          : 'text-ink-600 dark:text-ink-300 hover:bg-ink-50 dark:hover:bg-ink-800'}"
      >
        Next
        <ChevronRight class="w-4 h-4" />
      </button>
    </div>
  {/if}
{/if}
