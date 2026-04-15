<script lang="ts">
  import {
    getWatchFolderConfig,
    setWatchFolderConfig,
    listLibraries,
  } from "../../lib/api";
  import { required, validate } from "../../lib/validation";
  import { FolderSearch } from "lucide-svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import type { Library } from "../../types";

  let watchFolderPath = $state("");
  let watchFolderLibraryId = $state("");
  let libraries: Library[] = $state([]);
  let loading = $state(false);
  let error: string | null = $state(null);
  let successMessage: string | null = $state(null);

  let configured = $derived(watchFolderPath.trim() !== "");

  let submitLabel = $derived.by(() => {
    if (loading) return "Saving...";
    return configured ? "Update Configuration" : "Save Configuration";
  });

  $effect(() => {
    void (async () => {
      try {
        const [config, libs] = await Promise.all([
          getWatchFolderConfig(),
          listLibraries(),
        ]);
        watchFolderPath = config.path;
        watchFolderLibraryId = config.library_id;
        libraries = libs;
      } catch {
        // ignore - user can re-enter
      }
    })();
  });

  async function handleSave(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    successMessage = null;

    const trimmedPath = watchFolderPath.trim();

    // If path is set, library is required; if path is empty, we're clearing
    if (trimmedPath) {
      error =
        validate(trimmedPath, [required("Watch folder path is required")]) ??
        validate(watchFolderLibraryId, [
          required("Target library is required"),
        ]);
      if (error) return;
    }

    loading = true;

    try {
      const result = await setWatchFolderConfig({
        path: trimmedPath,
        library_id: trimmedPath ? watchFolderLibraryId : "",
      });
      successMessage = result.message;
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : "Failed to save watch folder configuration";
    } finally {
      loading = false;
    }
  }

  async function handleClear() {
    error = null;
    successMessage = null;
    loading = true;

    try {
      const result = await setWatchFolderConfig({
        path: "",
        library_id: "",
      });
      watchFolderPath = "";
      watchFolderLibraryId = "";
      successMessage = result.message;
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : "Failed to clear watch folder configuration";
    } finally {
      loading = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h2
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 flex items-center gap-2"
      >
        <FolderSearch class="w-5 h-5 text-accent-600" aria-hidden="true" />
        Watch Folder
      </h2>
    </div>

    <div class="mb-4">
      <div class="flex items-center gap-2 text-sm">
        <span class="text-ink-500 dark:text-ink-300">Status:</span>
        {#if configured}
          <span
            class="inline-flex items-center gap-1.5 text-success-700 dark:text-green-400 bg-success-50 dark:bg-green-900/20 px-2.5 py-1 rounded-full font-medium"
          >
            <span class="w-2 h-2 rounded-full bg-success-600"></span>
            Configured
          </span>
        {:else}
          <span
            class="inline-flex items-center gap-1.5 text-ink-600 dark:text-ink-300 bg-ink-50 dark:bg-ink-800 px-2.5 py-1 rounded-full font-medium"
          >
            <span class="w-2 h-2 rounded-full bg-ink-300"></span>
            Not configured
          </span>
        {/if}
      </div>
    </div>

    <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
      Configure a folder to watch for new book files. Any supported book files
      (.epub, .mobi, .pdf, .azw3) added to this folder will be automatically
      imported into the selected library.
    </p>

    <form onsubmit={handleSave} class="space-y-4">
      <div>
        <label
          for="watch-folder-path"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Folder Path
        </label>
        <TextInput
          id="watch-folder-path"
          type="text"
          bind:value={watchFolderPath}
          class="w-full py-2.5"
          placeholder="/path/to/watch/folder"
          disabled={loading}
          aria-describedby="watch-folder-path-hint"
        />
        <p
          id="watch-folder-path-hint"
          class="text-xs text-ink-500 dark:text-ink-300 mt-1"
        >
          An absolute path to a folder on the server. This folder will be
          scanned every minute for new book files.
        </p>
      </div>

      <div>
        <label
          for="watch-folder-library"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Target Library
        </label>
        <select
          id="watch-folder-library"
          bind:value={watchFolderLibraryId}
          class="w-full px-4 py-2.5 border border-ink-400 dark:border-ink-400 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent focus-visible:outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          disabled={loading}
          aria-describedby="watch-folder-library-hint"
        >
          <option value="">Select a library...</option>
          {#each libraries as lib (lib.id)}
            <option value={lib.id}>{lib.name}</option>
          {/each}
        </select>
        <p
          id="watch-folder-library-hint"
          class="text-xs text-ink-500 dark:text-ink-300 mt-1"
        >
          Books found in the watch folder will be added to this library
        </p>
      </div>

      {#if error}
        <AlertBanner variant="error">{error}</AlertBanner>
      {/if}

      {#if successMessage}
        <AlertBanner variant="success">{successMessage}</AlertBanner>
      {/if}

      <div class="flex gap-3">
        <Button type="submit" disabled={loading} class="flex-1">
          {submitLabel}
        </Button>
        {#if configured}
          <Button
            type="button"
            onclick={handleClear}
            disabled={loading}
            class="px-4 py-2.5 bg-ink-100 dark:bg-ink-700 text-ink-700 dark:text-ink-200 hover:bg-ink-200 dark:hover:bg-ink-600"
          >
            Clear
          </Button>
        {/if}
      </div>
    </form>
  </div>
</div>
