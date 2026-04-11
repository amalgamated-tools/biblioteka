<script lang="ts">
  import type { Book, RemoteMetadata } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import * as api from "../../lib/api";
  import { ArrowLeft, BookOpen } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import BookEditForm from "./BookEditForm.svelte";
  import type { FormFields } from "./BookEditForm.svelte";
  import MetadataFetchPanel from "./MetadataFetchPanel.svelte";

  interface Props {
    bookId: string;
  }

  let { bookId }: Props = $props();

  let book: Book | null = $state(null);
  let loading = $state(true);
  let error: string | null = $state(null);
  let saving = $state(false);
  let pendingMetadata = $state<RemoteMetadata | null>(null);

  let fields = $state<FormFields>({
    title: "",
    description: "",
    publisher: "",
    language: "",
    publicationDate: "",
    isbn13: "",
    isbn10: "",
    asin: "",
    goodreadsId: "",
    hardcoverId: "",
    googleBooksId: "",
    coverImageUrl: "",
  });

  let hasPendingMetadata = $derived(pendingMetadata !== null);

  $effect(() => {
    loadBook(bookId);
  });

  async function loadBook(id: string) {
    loading = true;
    error = null;
    try {
      book = await api.getBook(id);
      populateForm(book);
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load book";
    } finally {
      loading = false;
    }
  }

  function populateForm(b: Book) {
    fields.title = b.title;
    fields.description = b.description ?? "";
    fields.publisher = b.publisher ?? "";
    fields.language = b.language ?? "";
    fields.publicationDate = b.publication_date ?? "";
    fields.isbn13 = b.isbn13 ?? "";
    fields.isbn10 = b.isbn10 ?? "";
    fields.asin = b.asin ?? "";
    fields.goodreadsId = b.goodreads_id ?? "";
    fields.hardcoverId = b.hardcover_id ?? "";
    fields.googleBooksId = b.google_books_id ?? "";
    fields.coverImageUrl = b.cover_image_url ?? "";
  }
</script>

<div class="animate-fade-in">
  <div class="flex items-center gap-3 mb-6">
    <button
      onclick={() => routerStore.navigate(`books/${bookId}`)}
      class="text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
      aria-label="Back to book"
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
      Edit Book
    </h1>
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
  {:else}
    <MetadataFetchPanel {bookId} {saving} bind:fields bind:pendingMetadata />
    <BookEditForm
      {bookId}
      bind:fields
      {hasPendingMetadata}
      bind:saving
      onSaved={() => routerStore.navigate(`books/${bookId}`)}
    />
  {/if}
</div>
