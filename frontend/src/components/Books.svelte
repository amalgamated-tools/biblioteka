<script lang="ts">
  import { BookOpen, Search, X } from "lucide-svelte";
  import { untrack } from "svelte";
  import BookList from "./ui/BookList.svelte";
  import BookDetail from "./books/BookDetail.svelte";
  import BookEdit from "./books/BookEdit.svelte";
  import { routerStore } from "../stores/router.svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import * as api from "../lib/api";

  let initialOffset = $derived(
    Math.max(
      0,
      parseInt(routerStore.queryParams.get("offset") ?? "0", 10) || 0,
    ),
  );

  // Parse subPath: "", "{id}", "{id}/edit"
  let subParts = $derived(
    routerStore.subPath ? routerStore.subPath.split("/") : [],
  );
  let bookId = $derived(subParts[0] ?? "");
  let isEdit = $derived(subParts.length > 1 && subParts[1] === "edit");

  // Search state — input value (immediate) and debounced value (drives API calls).
  // `currentQuery` is derived from the URL so back/forward navigation and manual
  // hash edits are reflected in the input.
  let currentQuery = $derived(routerStore.queryParams.get("query") ?? "");
  let searchInput = $state(untrack(() => currentQuery));
  let debouncedQuery = $state(untrack(() => currentQuery));
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;

  // Sync local state when the URL query param changes externally.
  // Only track `currentQuery` — use untrack for local state comparisons
  // so that typing into the input doesn't trigger this effect.
  $effect(() => {
    const q = currentQuery;
    if (untrack(() => q === searchInput && q === debouncedQuery)) {
      return;
    }

    clearTimeout(debounceTimer);
    searchInput = q;
    debouncedQuery = q;
  });

  // Clean up any pending debounce timer when the component unmounts.
  $effect(() => {
    return () => {
      clearTimeout(debounceTimer);
      debounceTimer = undefined;
    };
  });

  function onSearchInput(value: string) {
    searchInput = value;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      debouncedQuery = value;
      routerStore.setQueryParam("query", value || null);
    }, 300);
  }

  function clearSearch() {
    searchInput = "";
    clearTimeout(debounceTimer);
    debouncedQuery = "";
    routerStore.setQueryParam("query", null);
  }

  // The derived `fetchBooks` function creates a new reference whenever
  // `debouncedQuery` changes. BookList's internal effect tracks `fetchBooks`
  // as a dependency, so it detects the new reference and resets pagination
  // offset to 0 before re-fetching.
  let fetchBooks = $derived((limit: number, offset: number) =>
    api.listBooks(limit, offset, debouncedQuery),
  );

  let emptyMessage = $derived(
    debouncedQuery ? `No results for "${debouncedQuery}"` : "No books yet.",
  );
  let emptySubMessage = $derived(
    debouncedQuery
      ? "Try a different search term."
      : "Books will appear here once they are added to your libraries.",
  );

  function handlePageChange(offset: number) {
    routerStore.setQueryParam("offset", offset === 0 ? null : String(offset));
  }
</script>

{#if bookId && isEdit}
  <BookEdit {bookId} />
{:else if bookId}
  <BookDetail {bookId} />
{:else}
  <div>
    <div class="flex items-center gap-3 mb-8">
      <div
        class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
      >
        <BookOpen
          class="w-5 h-5 text-accent-600 dark:text-accent-400"
          aria-hidden="true"
        />
      </div>
      <h1
        class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
      >
        All Books
      </h1>
    </div>

    <!-- Search input -->
    <div class="relative mb-6">
      <Search
        class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-400 dark:text-ink-500 pointer-events-none"
        aria-hidden="true"
      />
      <input
        type="search"
        value={searchInput}
        oninput={(e) => onSearchInput(e.currentTarget.value)}
        placeholder="Search books…"
        aria-label="Search books"
        class="w-full pl-9 pr-9 py-2 border border-ink-400 dark:border-ink-600 rounded-xl bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 placeholder:text-ink-400 dark:placeholder:text-ink-500 focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-transparent transition-all"
      />
      {#if searchInput}
        <button
          type="button"
          onclick={clearSearch}
          aria-label="Clear search"
          class="absolute right-3 top-1/2 -translate-y-1/2 text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
        >
          <X class="w-4 h-4" aria-hidden="true" />
        </button>
      {/if}
    </div>

    <!-- onBooksFound is intentionally omitted here. Books.svelte shows the aggregate
         view across all libraries, so it cannot know which library finished scanning.
         Each LibraryView calls clearScanning(libraryId) for its own library; the
         aggregate polling stops naturally once scanningIds empties. -->
    <BookList
      {fetchBooks}
      {initialOffset}
      onPageChange={handlePageChange}
      pollingInterval={libraryStore.isScanning ? 3000 : undefined}
      {emptyMessage}
      {emptySubMessage}
    />
  </div>
{/if}
