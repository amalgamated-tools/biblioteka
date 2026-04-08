<script lang="ts">
  import type { Book } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import * as api from "../../lib/api";
  import {
    BookOpen,
    ArrowLeft,
    Pencil,
    FileText,
    User,
    BookMarked,
  } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";

  interface Props {
    bookId: string;
  }

  let { bookId }: Props = $props();

  let book: Book | null = $state(null);
  let loading = $state(true);
  let error: string | null = $state(null);

  $effect(() => {
    loadBook(bookId);
  });

  async function loadBook(id: string) {
    loading = true;
    error = null;
    try {
      book = await api.getBook(id);
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load book";
    } finally {
      loading = false;
    }
  }
</script>

<div class="animate-fade-in">
  <div class="flex items-center gap-3 mb-6">
    <button
      onclick={() => routerStore.navigate("books")}
      class="text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
      aria-label="Back to books"
    >
      <ArrowLeft class="w-5 h-5" aria-hidden="true" />
    </button>
    <div
      class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
    >
      <BookOpen
        class="w-5 h-5 text-accent-600 dark:text-accent-400"
        aria-hidden="true"
      />
    </div>
    <h1
      class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100 truncate"
    >
      {book?.title ?? "Book"}
    </h1>
    {#if book}
      <button
        onclick={() => routerStore.navigate(`books/${bookId}/edit`)}
        class="ml-auto text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
        title="Edit book"
        aria-label="Edit book"
      >
        <Pencil class="w-5 h-5" aria-hidden="true" />
      </button>
    {/if}
  </div>

  {#if error}
    <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
  {:else if loading}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800"
    >
      <p role="status" class="text-center text-ink-500 dark:text-ink-300">
        Loading book...
      </p>
    </div>
  {:else if book}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- Cover & quick info -->
      <div
        class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
      >
        {#if book.cover_image_url}
          <img
            src={book.cover_image_url}
            alt={book.title}
            class="w-full rounded-xl mb-4"
          />
        {:else}
          <div
            class="aspect-[2/3] bg-ink-100 dark:bg-ink-800 rounded-xl flex items-center justify-center mb-4"
          >
            <BookOpen
              class="w-16 h-16 text-ink-300 dark:text-ink-600"
              aria-hidden="true"
            />
          </div>
        {/if}

        <Button
          onclick={() => routerStore.navigate(`books/${bookId}/edit`)}
          class="w-full py-2.5 text-sm"
        >
          <Pencil class="w-4 h-4 mr-1.5 inline" aria-hidden="true" />
          Edit Book
        </Button>
      </div>

      <!-- Details -->
      <div class="md:col-span-2 space-y-6">
        <div
          class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
        >
          <h2
            class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-4"
          >
            Details
          </h2>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-3 text-sm">
            {#each [
              ["Title", book.title],
              ["Publisher", book.publisher],
              ["Language", book.language],
              ["Publication Date", book.publication_date],
              ["ISBN-13", book.isbn13],
              ["ISBN-10", book.isbn10],
              ["ASIN", book.asin],
              ["Goodreads ID", book.goodreads_id],
              ["Hardcover ID", book.hardcover_id],
              ["Google Books ID", book.google_books_id],
            ] as [label, value] (label)}
              <div>
                <dt class="text-ink-500 dark:text-ink-400">{label}</dt>
                <dd class="text-ink-900 dark:text-cream-100 font-medium">
                  {value ?? "—"}
                </dd>
              </div>
            {/each}
          </dl>

          {#if book.description}
            <div class="mt-4 pt-4 border-t border-ink-100 dark:border-ink-800">
              <h3
                class="text-sm text-ink-500 dark:text-ink-400 mb-1"
              >
                Description
              </h3>
              <p
                class="text-sm text-ink-700 dark:text-ink-200 whitespace-pre-line"
              >
                {book.description}
              </p>
            </div>
          {/if}
        </div>

        <!-- Authors -->
        {#if book.authors.length > 0}
          <div
            class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
          >
            <h2
              class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-3"
            >
              <User class="w-4 h-4 inline mr-1.5" aria-hidden="true" />
              Authors
            </h2>
            <ul class="space-y-1 text-sm">
              {#each book.authors as author (author.id)}
                <li class="text-ink-700 dark:text-ink-200">{author.name}</li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Series -->
        {#if book.series.length > 0}
          <div
            class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
          >
            <h2
              class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-3"
            >
              <BookMarked class="w-4 h-4 inline mr-1.5" aria-hidden="true" />
              Series
            </h2>
            <ul class="space-y-1 text-sm">
              {#each book.series as entry (entry.series.id)}
                <li class="text-ink-700 dark:text-ink-200">
                  {entry.series.name}{entry.position != null
                    ? ` #${entry.position}`
                    : ""}
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Files -->
        {#if book.files.length > 0}
          <div
            class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
          >
            <h2
              class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-3"
            >
              <FileText class="w-4 h-4 inline mr-1.5" aria-hidden="true" />
              Files
            </h2>
            <ul class="space-y-2 text-sm">
              {#each book.files as file (file.id)}
                <li
                  class="flex items-center justify-between text-ink-700 dark:text-ink-200"
                >
                  <span class="truncate" title={file.file_path}
                    >{file.file_name}</span
                  >
                  <span
                    class="text-xs text-ink-400 dark:text-ink-500 uppercase ml-2 flex-shrink-0"
                    >{file.file_type}</span
                  >
                </li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
