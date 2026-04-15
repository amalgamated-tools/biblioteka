<script lang="ts">
  import { Sparkles } from "lucide-svelte";
  import { getRecommendations } from "../lib/api";
  import { routerStore } from "../stores/router.svelte";
  import type { BookSummary } from "../types";

  let books = $state<BookSummary[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  let fetched = false;
  $effect(() => {
    if (!fetched) {
      fetched = true;
      getRecommendations(10)
        .then((data) => {
          books = data;
        })
        .catch((err) => {
          console.error("Failed to fetch recommendations:", err);
          error =
            err instanceof Error
              ? err.message
              : "Failed to load recommendations";
        })
        .finally(() => {
          loading = false;
        });
    }
  });
</script>

<div
  class="mt-8 bg-white dark:bg-ink-900 rounded-2xl p-6 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in"
  data-testid="recommendations-section"
>
  <div class="flex items-center gap-2 mb-4">
    <Sparkles
      class="w-5 h-5 text-accent-600 dark:text-accent-400"
      aria-hidden="true"
    />
    <h2 class="text-xl font-display font-bold text-ink-900 dark:text-cream-100">
      You Might Also Like
    </h2>
  </div>

  {#if loading}
    <!-- Skeleton cards -->
    <ul
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3"
      aria-label="Loading recommendations"
      aria-busy="true"
    >
      {#each [0, 1, 2, 3, 4] as i (i)}
        <li
          class="rounded-xl bg-ink-100 dark:bg-ink-800 animate-pulse h-24"
        ></li>
      {/each}
    </ul>
  {:else if error}
    <p class="text-sm text-red-500 dark:text-red-400">{error}</p>
  {:else if books.length === 0}
    <p class="text-ink-500 dark:text-ink-300 leading-relaxed text-sm">
      Read some books to get personalized recommendations. Connect KOReader via
      <a
        href="#settings/kobo"
        class="text-accent-600 dark:text-accent-400 underline underline-offset-2 hover:text-accent-700 dark:hover:text-accent-300 transition-colors"
      >
        Settings → KOSync
      </a>
      to track your reading progress.
    </p>
  {:else}
    <ul
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3"
      aria-label="Recommended books"
    >
      {#each books as book (book.id)}
        <li>
          <button
            type="button"
            onclick={() => routerStore.navigate(`books/${book.id}`)}
            class="w-full text-left group rounded-xl border border-ink-100 dark:border-ink-800 bg-ink-50 dark:bg-ink-800/50 hover:border-accent-300 dark:hover:border-accent-700 hover:bg-accent-50 dark:hover:bg-accent-900/10 transition-all p-3 h-full flex flex-col gap-1.5"
            aria-label="View {book.title}"
          >
            {#if book.cover_image_url}
              <img
                src={book.cover_image_url}
                alt=""
                aria-hidden="true"
                class="w-full aspect-[2/3] object-cover rounded-lg mb-1"
                loading="lazy"
              />
            {:else}
              <div
                class="w-full aspect-[2/3] bg-gradient-to-br from-accent-100 to-accent-200 dark:from-accent-800/30 dark:to-accent-700/20 rounded-lg mb-1 flex items-center justify-center"
                aria-hidden="true"
              >
                <Sparkles
                  class="w-6 h-6 text-accent-400 dark:text-accent-600"
                  aria-hidden="true"
                />
              </div>
            {/if}
            <span
              class="text-xs font-medium text-ink-800 dark:text-cream-200 line-clamp-2 group-hover:text-accent-700 dark:group-hover:text-accent-300 transition-colors"
              title={book.title}
            >
              {book.title}
            </span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
