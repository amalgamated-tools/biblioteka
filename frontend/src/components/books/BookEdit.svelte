<script lang="ts">
  import type { Book, BookInput, RemoteMetadata } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import * as api from "../../lib/api";
  import { ArrowLeft, BookOpen, Search } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import MetadataComparison from "./MetadataComparison.svelte";

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

  // Metadata fetch state
  let metadata: RemoteMetadata | null = $state(null);
  let fetchingMetadata = $state(false);
  let metadataError: string | null = $state(null);
  let progressMessage: string | null = $state(null);
  let eventSource: EventSource | null = $state(null);

  $effect(() => {
    loadBook(bookId);
    return () => {
      eventSource?.close();
    };
  });

  async function loadBook(id: string) {
    loading = true;
    error = null;
    try {
      book = await api.getBook(id);
      populateForm(book);
      // Check for existing pending metadata
      try {
        metadata = await api.getMetadata(id);
      } catch {
        // No pending metadata, that's fine
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
      // Mark pending metadata as applied so it doesn't reappear on next visit.
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

  async function handleFetchMetadata() {
    fetchingMetadata = true;
    metadataError = null;
    progressMessage = "Starting metadata fetch...";

    try {
      // Open SSE connection first so no progress events are missed if the
      // worker processes the job before the subscription is established.
      eventSource?.close();
      const es = api.subscribeToMetadataEvents(bookId);
      eventSource = es;

      // Now trigger the job — the SSE subscription is already active.
      await api.fetchMetadata(bookId);

      // Client-side timeout: if SSE doesn't deliver a terminal event within
      // 60 seconds, close the connection and poll for results.
      const sseTimeout = setTimeout(() => {
        if (fetchingMetadata && eventSource === es) {
          es.close();
          eventSource = null;
          fetchingMetadata = false;
          progressMessage = null;
          loadPendingMetadata();
        }
      }, 60_000);

      function closeSSE() {
        clearTimeout(sseTimeout);
        es.close();
        eventSource = null;
      }

      es.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.message) {
            progressMessage = data.message;
          }
          if (data.event === "complete") {
            closeSSE();
            fetchingMetadata = false;
            progressMessage = null;
            loadPendingMetadata();
          } else if (data.event === "error") {
            closeSSE();
            fetchingMetadata = false;
            metadataError = data.message ?? "Metadata fetch failed";
            progressMessage = null;
          } else if (data.event === "not_found") {
            closeSSE();
            fetchingMetadata = false;
            metadataError = data.message ?? "No metadata found for this book";
            progressMessage = null;
          }
        } catch {
          // Ignore parse errors
        }
      };

      es.onerror = () => {
        closeSSE();
        fetchingMetadata = false;
        // If we got an error before any complete event, try loading metadata
        // in case the job finished before SSE connected
        loadPendingMetadata();
      };
    } catch (e) {
      fetchingMetadata = false;
      metadataError =
        e instanceof Error ? e.message : "Failed to start metadata fetch";
      progressMessage = null;
    }
  }

  async function loadPendingMetadata() {
    try {
      metadata = await api.getMetadata(bookId);
    } catch {
      // No metadata found
    }
  }

  function applyField(field: keyof RemoteMetadata) {
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
    if (metadata.title) title = metadata.title;
    if (metadata.description) description = metadata.description;
    if (metadata.publisher) publisher = metadata.publisher;
    if (metadata.language) language = metadata.language;
    if (metadata.publication_date) publicationDate = metadata.publication_date;
    if (metadata.isbn13) isbn13 = metadata.isbn13;
    if (metadata.isbn10) isbn10 = metadata.isbn10;
    if (metadata.asin) asin = metadata.asin;
    if (metadata.goodreads_id) goodreadsId = metadata.goodreads_id;
    if (metadata.hardcover_id) hardcoverId = metadata.hardcover_id;
    if (metadata.google_books_id) googleBooksId = metadata.google_books_id;
    if (metadata.cover_image_url) coverImageUrl = metadata.cover_image_url;
  }

  async function dismissMetadata() {
    if (!metadata) return;
    try {
      await api.rejectMetadata(bookId);
      metadata = null;
    } catch (e) {
      metadataError =
        e instanceof Error ? e.message : "Failed to dismiss metadata";
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
    <!-- Metadata fetch section -->
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 mb-6"
    >
      <div class="flex items-center justify-between mb-3">
        <h2
          class="text-lg font-display font-bold text-ink-900 dark:text-cream-100"
        >
          Remote Metadata
        </h2>
        <Button
          onclick={handleFetchMetadata}
          disabled={fetchingMetadata || saving}
          class="px-4 py-2 text-sm"
        >
          <Search class="w-4 h-4 mr-1.5 inline" aria-hidden="true" />
          {fetchingMetadata ? "Fetching..." : "Fetch Metadata"}
        </Button>
      </div>

      {#if fetchingMetadata && progressMessage}
        <div
          class="flex items-center gap-3 p-3 bg-accent-50 dark:bg-accent-900/20 rounded-xl text-sm text-accent-700 dark:text-accent-300"
        >
          <div
            class="w-4 h-4 rounded-full border-2 border-accent-300 dark:border-accent-600 border-t-accent-600 dark:border-t-accent-300 animate-spin flex-shrink-0"
            aria-hidden="true"
          ></div>
          <span aria-live="polite">{progressMessage}</span>
        </div>
      {/if}

      {#if metadataError}
        <AlertBanner variant="error" class="mt-3">{metadataError}</AlertBanner>
      {/if}

      {#if metadata && !fetchingMetadata}
        <MetadataComparison
          {metadata}
          currentValues={currentFormValues}
          onApplyField={applyField}
          onApplyAll={applyAll}
          onDismiss={dismissMetadata}
        />
      {/if}
    </div>

    <!-- Edit form -->
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
    >
      {#if formError}
        <AlertBanner variant="error" class="mb-4">{formError}</AlertBanner>
      {/if}

      <form onsubmit={handleSave} class="space-y-5">
        <div>
          <label
            for="book-title"
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
          >
            Title <span class="text-danger-600" aria-hidden="true">*</span>
          </label>
          <TextInput
            id="book-title"
            bind:value={title}
            placeholder="Book title"
            class="w-full py-2.5"
            disabled={saving}
            aria-required={true}
          />
        </div>

        <div>
          <label
            for="book-description"
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
          >
            Description
          </label>
          <textarea
            id="book-description"
            bind:value={description}
            placeholder="Book description"
            rows="3"
            class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 dark:placeholder-ink-500 transition-all resize-y"
            disabled={saving}
          ></textarea>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label
              for="book-publisher"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Publisher
            </label>
            <TextInput
              id="book-publisher"
              bind:value={publisher}
              placeholder="Publisher"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-language"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Language
            </label>
            <TextInput
              id="book-language"
              bind:value={language}
              placeholder="Language"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-pub-date"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Publication Date
            </label>
            <TextInput
              id="book-pub-date"
              bind:value={publicationDate}
              placeholder="YYYY-MM-DD"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-isbn13"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              ISBN-13
            </label>
            <TextInput
              id="book-isbn13"
              bind:value={isbn13}
              placeholder="ISBN-13"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-isbn10"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              ISBN-10
            </label>
            <TextInput
              id="book-isbn10"
              bind:value={isbn10}
              placeholder="ISBN-10"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-asin"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              ASIN
            </label>
            <TextInput
              id="book-asin"
              bind:value={asin}
              placeholder="ASIN"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-goodreads-id"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Goodreads ID
            </label>
            <TextInput
              id="book-goodreads-id"
              bind:value={goodreadsId}
              placeholder="Goodreads ID"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-hardcover-id"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Hardcover ID
            </label>
            <TextInput
              id="book-hardcover-id"
              bind:value={hardcoverId}
              placeholder="Hardcover ID"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-google-id"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Google Books ID
            </label>
            <TextInput
              id="book-google-id"
              bind:value={googleBooksId}
              placeholder="Google Books ID"
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>

          <div>
            <label
              for="book-cover-url"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >
              Cover Image URL
            </label>
            <TextInput
              id="book-cover-url"
              bind:value={coverImageUrl}
              placeholder="https://..."
              class="w-full py-2.5"
              disabled={saving}
            />
          </div>
        </div>

        <div class="flex items-center gap-3 pt-2">
          <Button
            type="submit"
            disabled={saving}
            class="px-5 py-2.5 text-sm active:scale-[0.98]"
          >
            {saving ? "Saving..." : "Save Changes"}
          </Button>
          <Button
            type="button"
            variant="secondary"
            onclick={() => routerStore.navigate(`books/${bookId}`)}
            disabled={saving}
            class="px-5 py-2.5 text-sm"
          >
            Cancel
          </Button>
        </div>
      </form>
    </div>
  {/if}
</div>
