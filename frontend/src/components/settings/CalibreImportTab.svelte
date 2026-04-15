<script lang="ts">
  import {
    previewCalibreImport,
    confirmCalibreImport,
    previewCalibreImportFromPath,
    confirmCalibreImportFromPath,
    listLibraries,
  } from "../../lib/api";
  import type {
    CalibrePreview,
    CalibreImportResult,
    Library,
  } from "../../types";
  import { DatabaseZap } from "lucide-svelte";
  import Button from "../ui/Button.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";

  type Step = "upload" | "preview" | "result";
  type ImportSource = "upload" | "path";

  const steps: Step[] = ["upload", "preview", "result"];
  const stepLabels: Record<Step, string> = {
    upload: "Upload",
    preview: "Preview",
    result: "Done",
  };

  let step: Step = $state("upload");
  let loading = $state(false);
  let error: string | null = $state(null);

  // Upload step state
  let importSource: ImportSource = $state("upload");
  let selectedFile: File | null = $state(null);
  let serverPath = $state("");
  let selectedLibraryId = $state("");
  let libraries: Library[] = $state([]);

  // Preview step state
  let preview: CalibrePreview | null = $state(null);

  // Result step state
  let result: CalibreImportResult | null = $state(null);

  $effect(() => {
    void (async () => {
      try {
        libraries = await listLibraries();
      } catch {
        // ignore — library selection is optional
      }
    })();
  });

  function handleFileChange(e: Event) {
    const input = e.target as HTMLInputElement;
    selectedFile = input.files?.[0] ?? null;
    error = null;
  }

  async function handlePreview(e: SubmitEvent) {
    e.preventDefault();

    if (importSource === "path") {
      if (!serverPath.trim()) {
        error = "Please enter a server path to metadata.db.";
        return;
      }
    } else {
      if (!selectedFile) {
        error = "Please select a Calibre metadata.db file.";
        return;
      }
    }

    error = null;
    loading = true;

    try {
      preview =
        importSource === "path"
          ? await previewCalibreImportFromPath(serverPath.trim())
          : await previewCalibreImport(selectedFile!);
      step = "preview";
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Failed to read Calibre database.";
    } finally {
      loading = false;
    }
  }

  async function handleConfirm() {
    if (importSource === "upload" && !selectedFile) return;
    if (importSource === "path" && !serverPath.trim()) return;

    error = null;
    loading = true;

    try {
      result =
        importSource === "path"
          ? await confirmCalibreImportFromPath(
              serverPath.trim(),
              selectedLibraryId,
            )
          : await confirmCalibreImport(selectedFile!, selectedLibraryId);
      step = "result";
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Import failed. Please try again.";
    } finally {
      loading = false;
    }
  }

  function handleReset() {
    step = "upload";
    importSource = "upload";
    selectedFile = null;
    serverPath = "";
    selectedLibraryId = "";
    preview = null;
    result = null;
    error = null;
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-1 flex items-center gap-2"
    >
      <DatabaseZap class="w-5 h-5 text-accent-600" aria-hidden="true" />
      Import from Calibre
    </h2>
    <p class="text-sm text-ink-500 dark:text-ink-300 mb-6">
      Migrate your Calibre library into Biblioteka. Upload your
      <code
        class="font-mono text-xs bg-ink-100 dark:bg-ink-800 px-1 py-0.5 rounded"
        >metadata.db</code
      >
      file or provide the path to it on the server, then preview and confirm the import.
    </p>

    <!-- Step indicators -->
    <nav aria-label="Import steps" class="flex items-center gap-2 mb-6">
      {#each steps as s, i (s)}
        {@const isActive = step === s}
        {@const isPast =
          (s === "upload" && (step === "preview" || step === "result")) ||
          (s === "preview" && step === "result")}
        <div class="flex items-center gap-2">
          {#if i > 0}
            <div
              class="h-px w-6 {isPast || isActive
                ? 'bg-accent-500'
                : 'bg-ink-200 dark:bg-ink-700'}"
            ></div>
          {/if}
          <span
            class="text-xs font-medium px-2.5 py-1 rounded-full {isActive
              ? 'bg-accent-500 text-white'
              : isPast
                ? 'bg-accent-100 text-accent-700 dark:bg-accent-900/30 dark:text-accent-400'
                : 'bg-ink-100 dark:bg-ink-800 text-ink-400 dark:text-ink-500'}"
            aria-current={isActive ? "step" : undefined}
          >
            {stepLabels[s]}
          </span>
        </div>
      {/each}
    </nav>

    <!-- Step 1: Upload -->
    {#if step === "upload"}
      <form onsubmit={handlePreview} class="space-y-4">
        <!-- Source toggle -->
        <fieldset>
          <legend
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >Import Source</legend
          >
          <div class="flex gap-2">
            <button
              type="button"
              onclick={() => {
                importSource = "upload";
                error = null;
              }}
              class="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors {importSource ===
              'upload'
                ? 'bg-accent-500 text-white'
                : 'bg-ink-100 dark:bg-ink-800 text-ink-600 dark:text-ink-300 hover:bg-ink-200 dark:hover:bg-ink-700'}"
              aria-pressed={importSource === "upload"}
            >
              Upload File
            </button>
            <button
              type="button"
              onclick={() => {
                importSource = "path";
                error = null;
              }}
              class="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors {importSource ===
              'path'
                ? 'bg-accent-500 text-white'
                : 'bg-ink-100 dark:bg-ink-800 text-ink-600 dark:text-ink-300 hover:bg-ink-200 dark:hover:bg-ink-700'}"
              aria-pressed={importSource === "path"}
            >
              Server Path
            </button>
          </div>
        </fieldset>

        {#if importSource === "upload"}
          <div>
            <label
              for="calibre-db-file"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Calibre metadata.db
            </label>
            <input
              id="calibre-db-file"
              type="file"
              accept=".db"
              onchange={handleFileChange}
              disabled={loading}
              class="block w-full text-sm text-ink-600 dark:text-ink-300
                file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0
                file:text-sm file:font-medium
                file:bg-accent-50 file:text-accent-700
                dark:file:bg-accent-900/30 dark:file:text-accent-400
                hover:file:bg-accent-100 dark:hover:file:bg-accent-900/50
                cursor-pointer"
              aria-describedby="calibre-db-hint"
            />
            <p
              id="calibre-db-hint"
              class="text-xs text-ink-500 dark:text-ink-300 mt-1"
            >
              The <code
                class="font-mono bg-ink-100 dark:bg-ink-800 px-1 py-0.5 rounded"
                >metadata.db</code
              > file is located in the root of your Calibre library folder. Maximum
              100 MB.
            </p>
          </div>
        {:else}
          <div>
            <label
              for="calibre-db-path"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Server path to metadata.db
            </label>
            <input
              id="calibre-db-path"
              type="text"
              bind:value={serverPath}
              disabled={loading}
              placeholder="/path/to/Calibre Library/metadata.db"
              class="w-full px-4 py-2.5 border border-ink-400 dark:border-ink-400 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
              aria-describedby="calibre-path-hint"
            />
            <p
              id="calibre-path-hint"
              class="text-xs text-ink-500 dark:text-ink-300 mt-1"
            >
              Enter the absolute filesystem path on the server where your
              Calibre <code
                class="font-mono bg-ink-100 dark:bg-ink-800 px-1 py-0.5 rounded"
                >metadata.db</code
              > is located.
            </p>
          </div>
        {/if}

        <div>
          <label
            for="calibre-library"
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
          >
            Target Library <span class="text-ink-400 font-normal"
              >(optional)</span
            >
          </label>
          <select
            id="calibre-library"
            bind:value={selectedLibraryId}
            disabled={loading}
            class="w-full px-4 py-2.5 border border-ink-400 dark:border-ink-400 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
            aria-describedby="calibre-library-hint"
          >
            <option value="">None (import without library)</option>
            {#each libraries as lib (lib.id)}
              <option value={lib.id}>{lib.name}</option>
            {/each}
          </select>
          <p
            id="calibre-library-hint"
            class="text-xs text-ink-500 dark:text-ink-300 mt-1"
          >
            Imported books will be added to this library. You can change this
            later.
          </p>
        </div>

        {#if error}
          <AlertBanner variant="error">{error}</AlertBanner>
        {/if}

        <Button
          type="submit"
          disabled={loading ||
            (importSource === "upload" ? !selectedFile : !serverPath.trim())}
          class="w-full px-4 py-2.5"
        >
          {loading ? "Reading database…" : "Preview Import"}
        </Button>
      </form>
    {/if}

    <!-- Step 2: Preview -->
    {#if step === "preview" && preview}
      <div class="space-y-4">
        <div
          class="rounded-xl border border-ink-100 dark:border-ink-700 bg-ink-50 dark:bg-ink-800/50 p-4"
        >
          <p class="text-sm text-ink-600 dark:text-ink-300">
            Found
            <span class="font-semibold text-ink-900 dark:text-cream-100"
              >{preview.total.toLocaleString()}</span
            >
            {preview.total === 1 ? "book" : "books"} in your Calibre library.
            {#if preview.total > preview.books.length}
              Showing the first {preview.books.length}.
            {/if}
          </p>
          <p class="text-xs text-ink-500 dark:text-ink-400 mt-1">
            Books already in Biblioteka (matched by ISBN, ASIN, or Goodreads ID)
            will be skipped automatically.
          </p>
        </div>

        {#if preview.books.length > 0}
          <div
            class="divide-y divide-ink-100 dark:divide-ink-700 rounded-xl border border-ink-100 dark:border-ink-700 overflow-hidden"
            role="list"
            aria-label="Preview of books to import"
          >
            {#each preview.books as book (book.calibre_id)}
              <div class="px-4 py-3 bg-white dark:bg-ink-900" role="listitem">
                <p
                  class="text-sm font-medium text-ink-900 dark:text-cream-100 truncate"
                >
                  {book.title}
                </p>
                <p
                  class="text-xs text-ink-500 dark:text-ink-400 mt-0.5 truncate"
                >
                  {#if book.authors.length > 0}
                    {book.authors.join(", ")}
                  {:else}
                    <span class="italic">No author</span>
                  {/if}
                  {#if book.series.length > 0}
                    &nbsp;·&nbsp;{book.series[0].name}
                  {/if}
                  {#if book.formats.length > 0}
                    &nbsp;·&nbsp;{book.formats.join(", ").toUpperCase()}
                  {/if}
                </p>
              </div>
            {/each}
          </div>
        {/if}

        {#if error}
          <AlertBanner variant="error">{error}</AlertBanner>
        {/if}

        <div class="flex gap-3">
          <Button
            type="button"
            onclick={handleConfirm}
            disabled={loading}
            class="flex-1 px-4 py-2.5"
          >
            {loading
              ? "Importing…"
              : `Import ${preview.total.toLocaleString()} ${preview.total === 1 ? "book" : "books"}`}
          </Button>
          <Button
            type="button"
            onclick={handleReset}
            disabled={loading}
            class="px-4 py-2.5 bg-ink-100 dark:bg-ink-700 text-ink-700 dark:text-ink-200 hover:bg-ink-200 dark:hover:bg-ink-600"
          >
            Start Over
          </Button>
        </div>
      </div>
    {/if}

    <!-- Step 3: Result -->
    {#if step === "result" && result}
      <div class="space-y-4">
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {#each [{ label: "Total", value: result.total, color: "text-ink-900 dark:text-cream-100" }, { label: "Imported", value: result.imported, color: "text-success-700 dark:text-green-400" }, { label: "Skipped", value: result.skipped, color: "text-ink-500 dark:text-ink-400" }, { label: "Errors", value: result.errors, color: result.errors > 0 ? "text-error-600 dark:text-red-400" : "text-ink-500 dark:text-ink-400" }] as stat (stat.label)}
            <div
              class="rounded-xl border border-ink-100 dark:border-ink-700 bg-ink-50 dark:bg-ink-800/50 p-3 text-center"
            >
              <p class="text-2xl font-bold {stat.color}">
                {stat.value.toLocaleString()}
              </p>
              <p class="text-xs text-ink-500 dark:text-ink-400 mt-0.5">
                {stat.label}
              </p>
            </div>
          {/each}
        </div>

        {#if result.errors > 0}
          <AlertBanner variant="error">
            {result.errors}
            {result.errors === 1 ? "book" : "books"} could not be imported due to
            errors. Check the server logs for details.
          </AlertBanner>
        {:else if result.imported > 0}
          <AlertBanner variant="success">
            Successfully imported {result.imported.toLocaleString()}
            {result.imported === 1 ? "book" : "books"} into Biblioteka.
          </AlertBanner>
        {:else}
          <AlertBanner variant="success">
            All books were already present in Biblioteka — nothing new to
            import.
          </AlertBanner>
        {/if}

        <Button
          type="button"
          onclick={handleReset}
          class="w-full px-4 py-2.5 bg-ink-100 dark:bg-ink-700 text-ink-700 dark:text-ink-200 hover:bg-ink-200 dark:hover:bg-ink-600"
        >
          Import Again
        </Button>
      </div>
    {/if}
  </div>
</div>
