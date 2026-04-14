<script lang="ts">
  import { BookOpen, Search } from "lucide-svelte";
  import BookList from "./ui/BookList.svelte";
  import BookDetail from "./books/BookDetail.svelte";
  import BookEdit from "./books/BookEdit.svelte";
  import { routerStore } from "../stores/router.svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import { debounce } from "../lib/debounce";
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

  let searchQuery = $state(routerStore.queryParams.get("query") ?? "");
  let debouncedQuery = $state(routerStore.queryParams.get("query") ?? "");

  const updateDebouncedQuery = debounce((v: string) => {
    debouncedQuery = v;
  }, 300);

  $effect(() => {
    updateDebouncedQuery(searchQuery);
  });

  $effect(() => {
    routerStore.setQueryParam("query", debouncedQuery || null);
  });

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

    <div class="relative mb-6">
      <Search
        class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-400 dark:text-ink-400 pointer-events-none"
        aria-hidden="true"
      />
      <input
        type="search"
        bind:value={searchQuery}
        placeholder="Search books…"
        aria-label="Search books"
        class="w-full pl-9 pr-4 py-2 border border-ink-400 dark:border-ink-400 rounded-xl bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 placeholder:text-ink-500 dark:placeholder:text-ink-300 focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-transparent transition-all"
      />
    </div>

    <!-- onBooksFound is intentionally omitted here. Books.svelte shows the aggregate
         view across all libraries, so it cannot know which library finished scanning.
         Each LibraryView calls clearScanning(libraryId) for its own library; the
         aggregate polling stops naturally once scanningIds empties. -->
    <BookList
      fetchBooks={api.listBooks}
      query={debouncedQuery}
      {initialOffset}
      onPageChange={handlePageChange}
      pollingInterval={libraryStore.isScanning ? 3000 : undefined}
    />
  </div>
{/if}
