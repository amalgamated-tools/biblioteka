<script lang="ts">
  import type { BookSummary, PaginatedBooks } from "../../types";
  import { BookOpen, LayoutGrid, List, ChevronLeft, ChevronRight } from "lucide-svelte";
  import AlertBanner from "./AlertBanner.svelte";
  import BookCard from "./BookCard.svelte";

  interface Props {
    fetchBooks: (limit: number, offset: number) => Promise<PaginatedBooks>;
    pageSize?: number;
  }

  let { fetchBooks, pageSize = 24 }: Props = $props();

  let books: BookSummary[] = $state([]);
  let total = $state(0);
  let offset = $state(0);
  let loading = $state(false);
  let error: string | null = $state(null);
  let viewMode: "grid" | "table" = $state("grid");
  let currentRequestId = 0;

  let totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));
  let currentPage = $derived(Math.floor(offset / pageSize) + 1);
  let rangeStart = $derived(total === 0 ? 0 : offset + 1);
  let rangeEnd = $derived(Math.min(offset + pageSize, total));

  async function load() {
    const requestId = ++currentRequestId;
    loading = true;
    error = null;
    try {
      const data = await fetchBooks(pageSize, offset);
      if (requestId !== currentRequestId) {
        return;
      }
      books = data.books;
      total = data.total;
    } catch (e) {
      if (requestId !== currentRequestId) {
        return;
      }
      error = e instanceof Error ? e.message : "Failed to load books";
      books = [];
      total = 0;
    } finally {
      if (requestId === currentRequestId) {
        loading = false;
      }
    }
  }

  $effect(() => {
    // Re-fetch when offset changes (including initial load)
    void offset;
    load();
  });

  function prevPage() {
    if (currentPage > 1) {
      offset = (currentPage - 2) * pageSize;
    }
  }

  function nextPage() {
    if (currentPage < totalPages) {
      offset = currentPage * pageSize;
    }
  }
</script>

{#if error}
  <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
{/if}

{#if loading}
  <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
    <div class="text-center py-8">
      <p class="text-ink-400 dark:text-ink-400">Loading books...</p>
    </div>
  </div>
{:else if !error && total === 0}
  <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
    <div class="text-center py-8">
      <BookOpen class="w-12 h-12 text-ink-200 dark:text-ink-700 mx-auto mb-4" />
      <p class="text-ink-400 dark:text-ink-400 text-lg">No books yet.</p>
      <p class="text-ink-300 dark:text-ink-500 text-sm mt-1">
        Books will appear here once they are added to your libraries.
      </p>
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
      >
        <List class="w-4 h-4" />
      </button>
    </div>
  </div>

  <!-- Grid View -->
  {#if viewMode === "grid"}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {#each books as book (book.id)}
        <BookCard {book} />
      {/each}
    </div>
  {:else}
    <!-- Table View -->
    <div class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-ink-100 dark:border-ink-800 text-left text-ink-400 dark:text-ink-500">
            <th class="px-4 py-3 font-medium">Title</th>
            <th class="px-4 py-3 font-medium hidden sm:table-cell">Publisher</th>
            <th class="px-4 py-3 font-medium hidden md:table-cell">Language</th>
            <th class="px-4 py-3 font-medium hidden md:table-cell text-right">Pages</th>
            <th class="px-4 py-3 font-medium hidden lg:table-cell">Published</th>
          </tr>
        </thead>
        <tbody>
          {#each books as book (book.id)}
            <tr class="border-b border-ink-50 dark:border-ink-800/50 hover:bg-ink-50 dark:hover:bg-ink-800/30 transition-colors">
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
                    <div class="w-8 h-12 bg-ink-100 dark:bg-ink-800 rounded flex items-center justify-center">
                      <BookOpen class="w-4 h-4 text-ink-300 dark:text-ink-600" />
                    </div>
                  {/if}
                  <span class="font-medium text-ink-900 dark:text-cream-100 truncate max-w-xs" title={book.title}>
                    {book.title}
                  </span>
                </div>
              </td>
              <td class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden sm:table-cell truncate max-w-[200px]">
                {book.publisher ?? "—"}
              </td>
              <td class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden md:table-cell">
                {book.language ?? "—"}
              </td>
              <td class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden md:table-cell text-right">
                {book.num_pages ?? "—"}
              </td>
              <td class="px-4 py-3 text-ink-500 dark:text-ink-400 hidden lg:table-cell">
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
