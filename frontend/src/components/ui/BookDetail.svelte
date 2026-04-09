<script lang="ts">
  import type { Book } from "../../types";
  import { getBook, bookFileDownloadUrl } from "../../lib/api";
  import { ArrowLeft, BookOpen, Download, FileText } from "lucide-svelte";
  import { routerStore } from "../../stores/router.svelte";
  import AlertBanner from "./AlertBanner.svelte";

  interface Props {
    bookId: string;
  }

  let { bookId }: Props = $props();

  let book: Book | null = $state(null);
  let loading = $state(true);
  let error: string | null = $state(null);

  $effect(() => {
    loading = true;
    error = null;
    book = null;
    getBook(bookId)
      .then((b) => {
        book = b;
      })
      .catch((e) => {
        error = e instanceof Error ? e.message : "Failed to load book";
      })
      .finally(() => {
        loading = false;
      });
  });

  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }
</script>

<div>
  <button
    onclick={() => routerStore.navigate("books")}
    class="flex items-center gap-1 text-sm text-ink-500 dark:text-ink-300 hover:text-ink-700 dark:hover:text-ink-100 transition-colors mb-6"
  >
    <ArrowLeft class="w-4 h-4" aria-hidden="true" />
    Back to books
  </button>

  {#if error}
    <AlertBanner variant="error">{error}</AlertBanner>
  {:else if loading}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800"
    >
      <p role="status" class="text-center text-ink-500 dark:text-ink-300">
        Loading book...
      </p>
    </div>
  {:else if book}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden"
    >
      <div class="p-6 md:p-8">
        <div class="flex flex-col md:flex-row gap-6">
          <!-- Cover image -->
          {#if book.cover_image_url}
            <div class="shrink-0">
              <img
                src={book.cover_image_url}
                alt={book.title}
                class="w-40 h-60 object-cover rounded-lg shadow-sm"
              />
            </div>
          {:else}
            <div
              class="shrink-0 w-40 h-60 bg-ink-100 dark:bg-ink-800 rounded-lg flex items-center justify-center"
            >
              <BookOpen
                class="w-12 h-12 text-ink-300 dark:text-ink-600"
                aria-hidden="true"
              />
            </div>
          {/if}

          <!-- Book info -->
          <div class="flex-1 min-w-0">
            <h1
              class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2"
            >
              {book.title}
            </h1>

            {#if book.authors.length > 0}
              <p class="text-ink-600 dark:text-ink-300 mb-3">
                by {book.authors.map((a) => a.name).join(", ")}
              </p>
            {/if}

            {#if book.series.length > 0}
              <p class="text-sm text-ink-500 dark:text-ink-400 mb-3">
                {#each book.series as entry, i (entry.series.id)}
                  {#if i > 0},
                  {/if}
                  {entry.series.name}{#if entry.position != null}
                    #{entry.position}{/if}
                {/each}
              </p>
            {/if}

            <div
              class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-ink-500 dark:text-ink-400 mb-4"
            >
              {#if book.publisher}
                <span>Publisher: {book.publisher}</span>
              {/if}
              {#if book.language}
                <span>Language: {book.language}</span>
              {/if}
              {#if book.publication_date}
                <span>Published: {book.publication_date}</span>
              {/if}
            </div>

            {#if book.description}
              <p
                class="text-sm text-ink-600 dark:text-ink-300 leading-relaxed line-clamp-4"
              >
                {book.description}
              </p>
            {/if}
          </div>
        </div>

        <!-- Files section -->
        {#if book.files.length > 0}
          <div class="mt-8">
            <h2
              class="text-lg font-display font-semibold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
            >
              <FileText class="w-5 h-5" aria-hidden="true" />
              Files
            </h2>
            <div class="space-y-3">
              {#each book.files as file (file.id)}
                <div
                  class="flex items-center justify-between gap-4 p-4 rounded-xl bg-ink-50 dark:bg-ink-800/50 border border-ink-100 dark:border-ink-800"
                >
                  <div class="min-w-0 flex-1">
                    <p
                      class="font-medium text-sm text-ink-900 dark:text-cream-100 truncate"
                      title={file.file_name}
                    >
                      {file.file_name}
                    </p>
                    <div
                      class="flex items-center gap-3 mt-1 text-xs text-ink-500 dark:text-ink-400"
                    >
                      <span
                        class="uppercase font-medium text-accent-600 dark:text-accent-400"
                        >{file.file_type}</span
                      >
                      <span>{formatFileSize(file.file_size)}</span>
                      <span
                        >{file.download_count}
                        {file.download_count === 1
                          ? "download"
                          : "downloads"}</span
                      >
                    </div>
                  </div>
                  <a
                    href={bookFileDownloadUrl(file.id)}
                    class="shrink-0 inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-accent-600 text-white hover:bg-accent-700 transition-colors"
                    download
                  >
                    <Download class="w-4 h-4" aria-hidden="true" />
                    Download
                  </a>
                </div>
              {/each}
            </div>
          </div>
        {:else}
          <div class="mt-8 text-center py-4">
            <p class="text-sm text-ink-500 dark:text-ink-300">
              No files available for this book.
            </p>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
