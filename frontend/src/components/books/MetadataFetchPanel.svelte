<script lang="ts">
  import type {
    RemoteMetadata,
    MetadataProgressEvent,
    CurrentValues,
  } from "../../types";
  import { ApiError } from "../../lib/api/core";
  import * as api from "../../lib/api";
  import { Search } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import MetadataComparison from "./MetadataComparison.svelte";
  import type { FormFields } from "./BookEditForm.svelte";

  // Maps each editable RemoteMetadata key to the corresponding FormFields key.
  // The `satisfies` constraint ensures this stays in sync with both interfaces at
  // compile time: adding or renaming a field in either interface will cause a
  // type error here.
  const FIELD_MAP = {
    title: "title",
    description: "description",
    publisher: "publisher",
    language: "language",
    publication_date: "publicationDate",
    isbn13: "isbn13",
    isbn10: "isbn10",
    asin: "asin",
    goodreads_id: "goodreadsId",
    hardcover_id: "hardcoverId",
    google_books_id: "googleBooksId",
    cover_image_url: "coverImageUrl",
  } as const satisfies Partial<Record<keyof RemoteMetadata, keyof FormFields>>;

  type EditableMetadataKey = keyof typeof FIELD_MAP;

  interface Props {
    bookId: string;
    saving: boolean;
    fields: FormFields;
    pendingMetadata?: RemoteMetadata | null;
  }

  let {
    bookId,
    saving,
    fields = $bindable() as FormFields,
    pendingMetadata = $bindable<RemoteMetadata | null>(null),
  }: Props = $props();

  let fetchingMetadata = $state(false);
  let applying = $state(false);
  let metadataError: string | null = $state(null);
  let progressMessage: string | null = $state(null);
  let eventSource: EventSource | null = null;
  let sseTimeout: ReturnType<typeof setTimeout> | null = null;

  let currentFormValues: CurrentValues = $derived({
    title: fields.title,
    description: fields.description || null,
    publisher: fields.publisher || null,
    language: fields.language || null,
    publication_date: fields.publicationDate || null,
    isbn13: fields.isbn13 || null,
    isbn10: fields.isbn10 || null,
    asin: fields.asin || null,
    goodreads_id: fields.goodreadsId || null,
    hardcover_id: fields.hardcoverId || null,
    google_books_id: fields.googleBooksId || null,
    cover_image_url: fields.coverImageUrl || null,
  });

  function closeSSE() {
    if (sseTimeout != null) {
      clearTimeout(sseTimeout);
      sseTimeout = null;
    }
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
  }

  // Close any open SSE connection and timeout when the component unmounts
  // or when bookId changes (e.g. navigating between edit pages without
  // remounting the component).
  $effect(() => {
    loadPendingMetadata(bookId);
    return () => {
      closeSSE();
      fetchingMetadata = false;
      applying = false;
      metadataError = null;
      progressMessage = null;
    };
  });

  async function loadPendingMetadata(forBookId?: string) {
    const targetBookId = forBookId ?? bookId;
    pendingMetadata = null;
    metadataError = null;
    try {
      const result = await api.getMetadata(targetBookId);
      // Only update state if bookId hasn't changed while the request was in-flight.
      if (bookId === targetBookId) {
        pendingMetadata = result;
      }
    } catch (e) {
      if (bookId !== targetBookId) return;
      pendingMetadata = null;
      // A 404 is expected when no pending metadata exists. Surface other
      // errors so the user knows something went wrong.
      if (!(e instanceof ApiError && e.status === 404)) {
        metadataError =
          e instanceof Error
            ? e.message
            : "Failed to check for pending metadata";
      }
    }
  }

  async function handleFetchMetadata() {
    fetchingMetadata = true;
    metadataError = null;
    progressMessage = "Starting metadata fetch...";

    try {
      // Open SSE connection first so no progress events are missed if the
      // worker processes the job before the subscription is established.
      // Fully reset any previous SSE connection and timeout from an earlier
      // fetch attempt before creating a new subscription.
      closeSSE();
      const es = api.subscribeToMetadataEvents(bookId);
      eventSource = es;

      // Client-side timeout: if SSE doesn't deliver a terminal event within
      // 60 seconds, close the connection and poll for results.
      sseTimeout = setTimeout(() => {
        if (fetchingMetadata && eventSource === es) {
          closeSSE();
          fetchingMetadata = false;
          progressMessage = null;
          loadPendingMetadata();
        }
      }, 60_000);

      // Register handlers immediately so no events are missed during the
      // fetch request that follows.
      es.onmessage = (event) => {
        // Guard against stale events from an EventSource that was replaced
        // (e.g. when bookId changed and $effect cleanup called closeSSE()).
        if (eventSource !== es) return;
        try {
          const data: MetadataProgressEvent = JSON.parse(event.data);
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
        // Guard against stale errors from a replaced EventSource.
        if (eventSource !== es) return;
        const capturedBookId = bookId;
        closeSSE();
        fetchingMetadata = false;
        // If we got an error before any complete event, try loading metadata
        // in case the job finished before SSE connected.
        loadPendingMetadata(capturedBookId).then(() => {
          // Bail if bookId changed while the request was in-flight.
          if (bookId !== capturedBookId) return;
          if (!pendingMetadata) {
            metadataError =
              "Metadata stream closed unexpectedly. Please try again.";
          }
        });
      };

      // Now trigger the job — the SSE subscription and handlers are already active.
      const response = await api.fetchMetadata(bookId);

      // If metadata already exists in the DB, no worker will publish SSE events.
      // Short-circuit to avoid waiting for the 60-second timeout.
      if (response.status === "already_exists") {
        closeSSE();
        fetchingMetadata = false;
        progressMessage = null;
        loadPendingMetadata();
        return;
      }

      // If a job is already in flight, keep the SSE subscription open so the
      // eventual complete/error event can still update the UI.
      if (response.status === "already_running") {
        progressMessage = "Metadata fetch already in progress...";
      }
    } catch (e) {
      closeSSE();
      fetchingMetadata = false;
      metadataError =
        e instanceof Error ? e.message : "Failed to start metadata fetch";
      progressMessage = null;
    }
  }

  async function dismissMetadata() {
    if (!pendingMetadata) return;
    try {
      await api.rejectMetadata(bookId);
      pendingMetadata = null;
    } catch (e) {
      metadataError =
        e instanceof Error ? e.message : "Failed to dismiss metadata";
    }
  }

  function applyField(field: EditableMetadataKey) {
    const meta = pendingMetadata;
    if (!meta) return;
    const value = meta[field];
    if (value == null) return;
    fields[FIELD_MAP[field]] = value;
  }

  async function applyAll() {
    if (!pendingMetadata) return;
    const targetBookId = bookId;
    applying = true;
    metadataError = null;
    try {
      const updated = await api.applyMetadata(targetBookId);
      if (bookId !== targetBookId) return;
      const src: Pick<typeof updated, EditableMetadataKey> = updated;
      for (const key of Object.keys(FIELD_MAP) as EditableMetadataKey[]) {
        fields[FIELD_MAP[key]] = src[key] ?? "";
      }
      pendingMetadata = null;
    } catch (e) {
      if (bookId !== targetBookId) return;
      metadataError =
        e instanceof Error ? e.message : "Failed to apply metadata";
    } finally {
      applying = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 mb-6"
>
  <div class="flex items-center justify-between mb-3">
    <h2 class="text-lg font-display font-bold text-ink-900 dark:text-cream-100">
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

  <p role="status" class="sr-only">{progressMessage ?? ""}</p>

  {#if fetchingMetadata && progressMessage}
    <div
      class="flex items-center gap-3 p-3 bg-accent-50 dark:bg-accent-900/20 rounded-xl text-sm text-accent-700 dark:text-accent-300"
      aria-hidden="true"
    >
      <div
        class="w-4 h-4 rounded-full border-2 border-accent-300 dark:border-accent-600 border-t-accent-600 dark:border-t-accent-300 animate-spin flex-shrink-0"
      ></div>
      <span>{progressMessage}</span>
    </div>
  {/if}

  {#if metadataError}
    <AlertBanner variant="error" class="mt-3">{metadataError}</AlertBanner>
  {/if}

  {#if pendingMetadata && !fetchingMetadata}
    <MetadataComparison
      metadata={pendingMetadata}
      currentValues={currentFormValues}
      onApplyField={applyField}
      onApplyAll={applyAll}
      onDismiss={dismissMetadata}
      {applying}
    />
  {/if}
</div>
