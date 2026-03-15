<script lang="ts">
  import type { Library, BookSummary } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import {
    Library as LibraryIcon,
    BookOpen,
    Settings2,
  } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import BookCard from "../ui/BookCard.svelte";

  interface Props {
    library: Library | null;
    libraryId: string;
    books: BookSummary[];
    loading: boolean;
    error: string | null;
  }

  let { library, libraryId, books, loading, error }: Props = $props();
</script>

<div class="animate-fade-in">
  <div class="flex items-center gap-3 mb-8">
    <div class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center">
      <LibraryIcon class="w-5 h-5 text-accent-600 dark:text-accent-400" />
    </div>
    <h1 class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100">
      {library?.name ?? "Library"}
    </h1>
    <button
      onclick={() => routerStore.navigate(`libraries/edit/${libraryId}`)}
      class="ml-auto text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
      title="Library settings"
      aria-label="Library settings"
    >
      <Settings2 class="w-5 h-5" />
    </button>
  </div>

  {#if error}
    <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
  {/if}

  {#if loading}
    <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
      <div class="text-center py-8">
        <p class="text-ink-400 dark:text-ink-400">Loading books...</p>
      </div>
    </div>
  {:else if !error && books.length === 0}
    <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
      <div class="text-center py-8">
        <BookOpen class="w-12 h-12 text-ink-200 dark:text-ink-700 mx-auto mb-4" />
        <p class="text-ink-400 dark:text-ink-400 text-lg">No books yet.</p>
        <p class="text-ink-300 dark:text-ink-500 text-sm mt-1">
          Books will appear here once they are scanned from your library folders.
        </p>
      </div>
    </div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {#each books as book (book.id)}
        <BookCard {book} />
      {/each}
    </div>
  {/if}
</div>
