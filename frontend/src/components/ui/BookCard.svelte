<script lang="ts">
  import type { BookSummary } from "../../types";
  import { BookOpen, Mail } from "lucide-svelte";
  import EmailBookModal from "./EmailBookModal.svelte";

  interface Props {
    book: BookSummary;
  }

  let { book }: Props = $props();

  let showEmailModal = $state(false);
</script>

<div
  class="relative bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden hover:shadow-md transition-shadow"
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

  <!-- Email button: always visible, accent on hover/focus -->
  <button
    onclick={() => (showEmailModal = true)}
    aria-label="Email {book.title}"
    title="Email this book"
    class="absolute top-2 right-2 p-1.5 rounded-lg bg-white/80 dark:bg-ink-900/80 backdrop-blur-sm text-ink-400 hover:text-accent-600 dark:hover:text-accent-400 focus:text-accent-600 dark:focus:text-accent-400 transition-colors shadow-sm border border-ink-100 dark:border-ink-700"
  >
    <Mail class="w-4 h-4" aria-hidden="true" />
  </button>
</div>

{#if showEmailModal}
  <EmailBookModal bookId={book.id} onClose={() => (showEmailModal = false)} />
{/if}
