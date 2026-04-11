<script lang="ts">
  import type { Book, BookInput, RemoteMetadata } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import * as api from "../../lib/api";
  import { ArrowLeft, BookOpen } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import MetadataFetchPanel from "./MetadataFetchPanel.svelte";
  import BookEditForm from "./BookEditForm.svelte";

  interface Props {
    bookId: string;
  }

  let { bookId }: Props = $props();

  let book: Book | null = $state(null);
  let loading = $state(true);
  let saving = $state(false);
  let error: string | null = $state(null);
  let formError: string | null = $state(null);

  // Form fields
  let title = $state("");
  let description = $state("");
  let publisher = $state("");
  let language = $state("");
  let publicationDate = $state("");
  let isbn13 = $state("");
  let isbn10 = $state("");
  let asin = $state("");
  let goodreadsId = $state("");
  let hardcoverId = $state("");
  let googleBooksId = $state("");
  let coverImageUrl = $state("");

  // Pending remote metadata (shared with MetadataFetchPanel via bind:)
  let metadata: RemoteMetadata | null = $state(null);

  $effect(() => {
    loadBook(bookId);
  });

  async function loadBook(id: string) {
    loading = true;
    error = null;
    metadata = null;
    try {
      book = await api.getBook(id);
      populateForm(book);
      // Check for existing pending metadata (404 means none exists;
      // other errors are suppressed — the user can re-fetch if needed)
      try {
        metadata = await api.getMetadata(id);
      } catch {
        metadata = null;
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load book";
    } finally {
      loading = false;
    }
  }

  function populateForm(b: Book) {
    title = b.title;
    description = b.description ?? "";
    publisher = b.publisher ?? "";
    language = b.language ?? "";
    publicationDate = b.publication_date ?? "";
    isbn13 = b.isbn13 ?? "";
    isbn10 = b.isbn10 ?? "";
    asin = b.asin ?? "";
    goodreadsId = b.goodreads_id ?? "";
    hardcoverId = b.hardcover_id ?? "";
    googleBooksId = b.google_books_id ?? "";
    coverImageUrl = b.cover_image_url ?? "";
  }

  async function handleSave(e: SubmitEvent) {
    e.preventDefault();
    if (!title.trim()) {
      formError = "Title is required";
      return;
    }

    saving = true;
    formError = null;
    try {
      const input: BookInput = {
        title: title.trim(),
        description: description.trim() || null,
        publisher: publisher.trim() || null,
        language: language.trim() || null,
        publication_date: publicationDate.trim() || null,
        isbn13: isbn13.trim() || null,
        isbn10: isbn10.trim() || null,
        asin: asin.trim() || null,
        goodreads_id: goodreadsId.trim() || null,
        hardcover_id: hardcoverId.trim() || null,
        google_books_id: googleBooksId.trim() || null,
        cover_image_url: coverImageUrl.trim() || null,
      };
      await api.updateBook(bookId, input);
      // Clear pending metadata so it doesn't reappear on next visit.
      if (metadata) {
        try {
          await api.rejectMetadata(bookId);
        } catch {
          // Best effort — the book is already saved.
        }
      }
      routerStore.navigate(`books/${bookId}`);
    } catch (e) {
      formError = e instanceof Error ? e.message : "Failed to save book";
    } finally {
      saving = false;
    }
  }

  let currentFormValues = $derived({
    title,
    description: description || null,
    publisher: publisher || null,
    language: language || null,
    publication_date: publicationDate || null,
    isbn13: isbn13 || null,
    isbn10: isbn10 || null,
    asin: asin || null,
    goodreads_id: goodreadsId || null,
    hardcover_id: hardcoverId || null,
    google_books_id: googleBooksId || null,
    cover_image_url: coverImageUrl || null,
  });

  type EditableMetadataField = keyof RemoteMetadata &
    keyof typeof currentFormValues;

  function applyField(field: EditableMetadataField) {
    if (!metadata) return;
    const value = metadata[field];
    if (value == null) return;

    const fieldStr = String(value);
    switch (field) {
      case "title":
        title = fieldStr;
        break;
      case "description":
        description = fieldStr;
        break;
      case "publisher":
        publisher = fieldStr;
        break;
      case "language":
        language = fieldStr;
        break;
      case "publication_date":
        publicationDate = fieldStr;
        break;
      case "isbn13":
        isbn13 = fieldStr;
        break;
      case "isbn10":
        isbn10 = fieldStr;
        break;
      case "asin":
        asin = fieldStr;
        break;
      case "goodreads_id":
        goodreadsId = fieldStr;
        break;
      case "hardcover_id":
        hardcoverId = fieldStr;
        break;
      case "google_books_id":
        googleBooksId = fieldStr;
        break;
      case "cover_image_url":
        coverImageUrl = fieldStr;
        break;
    }
  }

  function applyAll() {
    if (!metadata) return;
    if (metadata.title != null) title = metadata.title;
    if (metadata.description != null) description = metadata.description;
    if (metadata.publisher != null) publisher = metadata.publisher;
    if (metadata.language != null) language = metadata.language;
    if (metadata.publication_date != null)
      publicationDate = metadata.publication_date;
    if (metadata.isbn13 != null) isbn13 = metadata.isbn13;
    if (metadata.isbn10 != null) isbn10 = metadata.isbn10;
    if (metadata.asin != null) asin = metadata.asin;
    if (metadata.goodreads_id != null) goodreadsId = metadata.goodreads_id;
    if (metadata.hardcover_id != null) hardcoverId = metadata.hardcover_id;
    if (metadata.google_books_id != null)
      googleBooksId = metadata.google_books_id;
    if (metadata.cover_image_url != null)
      coverImageUrl = metadata.cover_image_url;
  }

  async function dismissMetadata() {
    if (!metadata) return;
    try {
      await api.rejectMetadata(bookId);
      metadata = null;
    } catch {
      // Best effort — if rejection fails, metadata stays visible and the user can retry.
    }
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
    <MetadataFetchPanel
      {bookId}
      {saving}
      bind:metadata
      currentValues={currentFormValues}
      onApplyField={applyField}
      onApplyAll={applyAll}
      onDismiss={dismissMetadata}
    />

    <BookEditForm
      bind:title
      bind:description
      bind:publisher
      bind:language
      bind:publicationDate
      bind:isbn13
      bind:isbn10
      bind:asin
      bind:goodreadsId
      bind:hardcoverId
      bind:googleBooksId
      bind:coverImageUrl
      {saving}
      {formError}
      onsubmit={handleSave}
      oncancel={() => routerStore.navigate(`books/${bookId}`)}
    />
  {/if}
</div>
