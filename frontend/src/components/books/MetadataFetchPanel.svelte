<script lang="ts">
  import type { RemoteMetadata, MetadataProgressEvent } from "../../types";
  import * as api from "../../lib/api";
  import { Search } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import MetadataComparison from "./MetadataComparison.svelte";

  interface CurrentValues {
    title: string;
    description: string | null;
    publisher: string | null;
    language: string | null;
    publication_date: string | null;
    isbn13: string | null;
    isbn10: string | null;
    asin: string | null;
    goodreads_id: string | null;
    hardcover_id: string | null;
    google_books_id: string | null;
    cover_image_url: string | null;
  }

  type EditableMetadataField = keyof RemoteMetadata & keyof CurrentValues;

  interface Props {
    bookId: string;
    saving: boolean;
    metadata: RemoteMetadata | null;
    currentValues: CurrentValues;
    onApplyField: (field: EditableMetadataField) => void;
    onApplyAll: () => void;
    onDismiss: () => void;
  }

  let {
    bookId,
    saving,
    metadata = $bindable(null),
    currentValues,
    onApplyField,
    onApplyAll,
    onDismiss,
  }: Props = $props();

  let fetchingMetadata = $state(false);
  let metadataError: string | null = $state(null);
  let progressMessage: string | null = $state(null);
  let eventSource: EventSource | null = null;
  let sseTimeout: ReturnType<typeof setTimeout> | null = null;

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
    // Read bookId so Svelte tracks it as a dependency.
    void bookId;
    return () => {
      closeSSE();
      fetchingMetadata = false;
      metadataError = null;
      progressMessage = null;
    };
  });

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
        closeSSE();
        fetchingMetadata = false;
        // If we got an error before any complete event, try loading metadata
        // in case the job finished before SSE connected.
        loadPendingMetadata().then(() => {
          if (!metadata) {
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

  async function loadPendingMetadata() {
    try {
      metadata = await api.getMetadata(bookId);
    } catch {
      metadata = null;
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
      {currentValues}
      {onApplyField}
      {onApplyAll}
      {onDismiss}
    />
  {/if}
</div>
