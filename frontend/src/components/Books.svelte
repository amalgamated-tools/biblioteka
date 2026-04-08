<script lang="ts">
  import { BookOpen } from "lucide-svelte";
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
  let bookId = $derived.by(() => {
    const sub = routerStore.subPath;
    if (!sub) return "";
    const parts = sub.split("/");
    return parts[0];
  });

  let isEdit = $derived.by(() => {
    const sub = routerStore.subPath;
    if (!sub) return false;
    const parts = sub.split("/");
    return parts.length > 1 && parts[1] === "edit";
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

    <!-- onBooksFound is intentionally omitted here. Books.svelte shows the aggregate
         view across all libraries, so it cannot know which library finished scanning.
         Each LibraryView calls clearScanning(libraryId) for its own library; the
         aggregate polling stops naturally once scanningIds empties. -->
    <BookList
      fetchBooks={api.listBooks}
      {initialOffset}
      onPageChange={handlePageChange}
      pollingInterval={libraryStore.isScanning ? 3000 : undefined}
    />
  </div>
{/if}
