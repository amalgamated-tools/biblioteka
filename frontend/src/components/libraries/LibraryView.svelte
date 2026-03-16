<script lang="ts">
  import type { Library } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import {
    Library as LibraryIcon,
    Settings2,
  } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import BookList from "../ui/BookList.svelte";
  import * as api from "../../lib/api";

  interface Props {
    library: Library | null;
    libraryId: string;
    error: string | null;
  }

  let { library, libraryId, error }: Props = $props();

  function fetchBooks(limit: number, offset: number) {
    return api.listLibraryBooks(libraryId, limit, offset);
  }
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

  {#key libraryId}
    <BookList {fetchBooks} />
  {/key}
</div>
