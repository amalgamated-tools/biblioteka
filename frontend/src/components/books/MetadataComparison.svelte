<script lang="ts">
  import type { RemoteMetadata, CurrentValues } from "../../types";
  import { ArrowLeft, Check, X } from "lucide-svelte";
  import Button from "../ui/Button.svelte";

  interface Props {
    metadata: RemoteMetadata;
    currentValues: CurrentValues;
    onApplyField: (field: keyof RemoteMetadata & keyof CurrentValues) => void;
    onApplyAll: () => void;
    onDismiss: () => void;
    applying?: boolean;
  }

  let {
    metadata,
    currentValues,
    onApplyField,
    onApplyAll,
    onDismiss,
    applying = false,
  }: Props = $props();

  type FieldDef = {
    key: keyof RemoteMetadata & keyof CurrentValues;
    label: string;
  };

  const fields: FieldDef[] = [
    { key: "title", label: "Title" },
    { key: "description", label: "Description" },
    { key: "publisher", label: "Publisher" },
    { key: "language", label: "Language" },
    { key: "publication_date", label: "Publication Date" },
    { key: "isbn13", label: "ISBN-13" },
    { key: "isbn10", label: "ISBN-10" },
    { key: "asin", label: "ASIN" },
    { key: "goodreads_id", label: "Goodreads ID" },
    { key: "hardcover_id", label: "Hardcover ID" },
    { key: "google_books_id", label: "Google Books ID" },
    { key: "cover_image_url", label: "Cover Image URL" },
  ];

  function displayValue(value: string | null | undefined): string {
    return value ?? "—";
  }

  function isDifferent(
    key: keyof RemoteMetadata & keyof CurrentValues,
  ): boolean {
    const remote = metadata[key];
    const current = currentValues[key];
    if (remote == null) return false;
    return String(remote) !== (current ?? "");
  }

  function sourceLabel(source: string): string {
    switch (source) {
      case "goodreads":
        return "Goodreads";
      default:
        return source;
    }
  }
</script>

<div class="mt-4 pt-4 border-t border-ink-100 dark:border-ink-800">
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
      <h3 class="text-sm font-semibold text-ink-700 dark:text-ink-200">
        Fetched Metadata
      </h3>
      <span
        class="text-xs px-2 py-0.5 rounded-full bg-accent-100 dark:bg-accent-800/30 text-accent-700 dark:text-accent-300"
      >
        {sourceLabel(metadata.source)}
      </span>
    </div>
    <div class="flex items-center gap-2">
      <Button onclick={onApplyAll} size="sm" disabled={applying}>
        <Check class="w-3 h-3 mr-1" aria-hidden="true" />
        {applying ? "Applying..." : "Apply All"}
      </Button>
      <Button
        variant="secondary"
        onclick={onDismiss}
        size="sm"
        disabled={applying}
      >
        <X class="w-3 h-3 mr-1" aria-hidden="true" />
        Dismiss
      </Button>
    </div>
  </div>

  <table
    class="w-full table-fixed border-separate border-spacing-x-0 border-spacing-y-2"
  >
    <caption class="sr-only">Comparison of current and fetched metadata</caption
    >
    <thead class="sr-only">
      <tr>
        <th scope="col">Field</th>
        <th scope="col">Current Value</th>
        <th scope="col">Action</th>
        <th scope="col">Fetched Value</th>
      </tr>
    </thead>
    <tbody>
      {#each fields as field (field.key)}
        {@const remote = metadata[field.key]}
        {#if remote != null}
          {@const different = isDifferent(field.key)}
          {@const cellBg = different
            ? "bg-accent-50 dark:bg-accent-800/20"
            : "bg-ink-50 dark:bg-ink-800/30"}
          <tr class="text-sm">
            <th
              scope="row"
              class="w-[120px] text-left text-ink-500 dark:text-ink-400 font-medium text-xs pt-0.5 p-2 rounded-l-lg align-top {cellBg} {different
                ? 'border-l-2 border-accent-500'
                : ''}"
            >
              {field.label}
            </th>
            <td
              class="text-ink-700 dark:text-ink-200 truncate p-2 align-top {cellBg}"
              title={displayValue(currentValues[field.key])}
            >
              {displayValue(currentValues[field.key])}
            </td>
            <td class="w-12 p-2 align-top {cellBg}">
              <div class="flex items-center justify-center">
                {#if different}
                  <button
                    onclick={() => onApplyField(field.key)}
                    disabled={applying}
                    class="p-1 rounded-md text-accent-600 dark:text-accent-400 hover:bg-accent-100 dark:hover:bg-accent-800/30 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                    title="Use fetched value"
                    aria-label="Use fetched {field.label}"
                  >
                    <ArrowLeft class="w-3.5 h-3.5" aria-hidden="true" />
                  </button>
                {:else}
                  <span
                    class="p-1 text-success-500 dark:text-success-400"
                    title="Values match"
                    aria-label={`Values match for ${field.label}`}
                    role="img"
                  >
                    <Check class="w-3.5 h-3.5" aria-hidden="true" />
                  </span>
                {/if}
              </div>
            </td>
            <td
              class="truncate p-2 rounded-r-lg align-top {cellBg} {different
                ? 'text-accent-700 dark:text-accent-300 font-medium'
                : 'text-ink-500 dark:text-ink-400'}"
              title={String(remote)}
            >
              {String(remote)}
            </td>
          </tr>
        {/if}
      {/each}
    </tbody>
  </table>
</div>
