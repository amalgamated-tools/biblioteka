<script lang="ts">
  import type { BookSummary } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import { BookOpen } from "lucide-svelte";

  interface Props {
    book: BookSummary;
  }

  let { book }: Props = $props();

  function handleClick(e: MouseEvent) {
    // Preserve native link semantics for modified clicks (open in new tab, etc.)
    if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
      return;
    }
    e.preventDefault();
    routerStore.navigate(`books/${book.id}`);
  }
</script>

<a
  href={`#books/${book.id}`}
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden hover:shadow-md transition-shadow cursor-pointer block no-underline"
  onclick={handleClick}
>
  {#if book.cover_image_url}
    <div class="aspect-[2/3] bg-ink-100 dark:bg-ink-800">
      <img
        src={book.cover_image_url}
        alt={book.title}
        loading="lazy"
        class="w-full h-full object-cover"
      />
    </div>
  {:else}
    <div
      class="aspect-[2/3] bg-ink-100 dark:bg-ink-800 flex items-center justify-center"
    >
      <BookOpen
        class="w-10 h-10 text-ink-300 dark:text-ink-600"
        aria-hidden="true"
      />
    </div>
  {/if}
  <div class="p-3">
    <h3
      class="font-medium text-sm text-ink-900 dark:text-cream-100 truncate"
      title={book.title}
    >
      {book.title}
    </h3>
    {#if book.publisher}
      <p class="text-xs text-ink-500 dark:text-ink-300 truncate mt-0.5">
        {book.publisher}
      </p>
    {/if}
  </div>
</a>
