<script lang="ts">
  import { onMount } from "svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import { routerStore } from "../stores/router.svelte";
  import * as api from "../lib/api";
  import type { BookSummary } from "../types";
  import {
    Plus,
    FolderOpen,
    Trash2,
    X,
    Library as LibraryIcon,
    BookOpen,
    Settings2,
  } from "lucide-svelte";

  let error: string | null = $state(null);
  let saving = $state(false);

  // Form state
  let editingId: string | null = $state(null);
  let formName = $state("");
  let nextPathId = 0;
  let formPaths: { id: number; value: string }[] = $state([
    { id: nextPathId++, value: "" },
  ]);
  let formMonitored = $state(false);
  let formError: string | null = $state(null);

  // Delete confirmation
  let showDeleteConfirm = $state(false);

  // Library view state
  let viewBooks: BookSummary[] = $state([]);
  let viewLoading = $state(false);

  // Determine mode from subPath: "new", "edit/{id}", "{id}" (view), or empty
  let mode: "create" | "edit" | "view" | "empty" = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp === "new") return "create";
    if (sp.startsWith("edit/")) return "edit";
    if (sp !== "") return "view";
    return "empty";
  });

  let editId: string = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp.startsWith("edit/")) return sp.slice(5);
    return "";
  });

  let viewId: string = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp === "new" || sp.startsWith("edit/") || sp === "") return "";
    return sp;
  });

  let viewLibrary = $derived.by(() => {
    if (!viewId) return null;
    return libraryStore.libraries.find((l) => l.id === viewId) ?? null;
  });

  // React to mode changes and library data arriving
  $effect(() => {
    if (mode === "create") {
      editingId = null;
      formName = "";
      formPaths = [{ id: nextPathId++, value: "" }];
      formMonitored = false;
      formError = null;
      showDeleteConfirm = false;
    } else if (mode === "edit" && editId) {
      const lib = libraryStore.libraries.find((l) => l.id === editId);
      if (lib) {
        editingId = lib.id;
        formName = lib.name;
        formPaths =
          lib.paths.length > 0
            ? lib.paths.map((p) => ({ id: nextPathId++, value: p }))
            : [{ id: nextPathId++, value: "" }];
        formMonitored = lib.monitored;
        formError = null;
        showDeleteConfirm = false;
      }
    } else if (mode === "view" && viewId) {
      loadLibraryBooks(viewId);
    }
  });

  async function loadLibraryBooks(libraryId: string) {
    viewLoading = true;
    error = null;
    try {
      const books = await api.listLibraryBooks(libraryId);
      if (viewId === libraryId) {
        viewBooks = books;
      }
    } catch (e) {
      if (viewId === libraryId) {
        error = e instanceof Error ? e.message : "Failed to load books";
        viewBooks = [];
      }
    } finally {
      if (viewId === libraryId) {
        viewLoading = false;
      }
    }
  }

  onMount(async () => {
    if (!libraryStore.loaded) {
      try {
        await libraryStore.load();
      } catch (e) {
        error = e instanceof Error ? e.message : "Failed to load libraries";
      }
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    formError = null;

    const name = formName.trim();
    if (!name) {
      formError = "Name is required";
      return;
    }

    const paths = formPaths
      .map((entry) => entry.value.trim())
      .filter((p) => p.length > 0);

    if (paths.length === 0) {
      formError = "At least one folder is required";
      return;
    }

    saving = true;
    try {
      const input = {
        name,
        paths,
        organization_type: "book_per_folder",
        monitored: formMonitored,
      };

      if (editingId) {
        await libraryStore.edit(editingId, input);
        routerStore.navigate(`libraries/${editingId}`);
      } else {
        const lib = await libraryStore.add(input);
        routerStore.navigate(`libraries/${lib.id}`);
      }
    } catch (e) {
      formError = e instanceof Error ? e.message : "Failed to save library";
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!editingId) return;
    try {
      await libraryStore.remove(editingId);
      routerStore.navigate("libraries");
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to delete library";
    }
  }
</script>

<div>
  {#if error}
    <div
      class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm mb-4 animate-scale-in"
    >
      {error}
    </div>
  {/if}

  {#if mode === "view"}
    <div class="animate-fade-in">
      <div class="flex items-center gap-3 mb-8">
        <div class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center">
          <LibraryIcon class="w-5 h-5 text-accent-600 dark:text-accent-400" />
        </div>
        <h1 class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100">
          {viewLibrary?.name ?? "Library"}
        </h1>
        <button
          onclick={() => routerStore.navigate(`libraries/edit/${viewId}`)}
          class="ml-auto text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors"
          title="Library settings"
          aria-label="Library settings"
        >
          <Settings2 class="w-5 h-5" />
        </button>
      </div>

      {#if viewLoading}
        <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
          <div class="text-center py-8">
            <p class="text-ink-400 dark:text-ink-400">Loading books...</p>
          </div>
        </div>
      {:else if viewBooks.length === 0}
        <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
          <div class="text-center py-8">
            <BookOpen class="w-12 h-12 text-ink-200 dark:text-ink-700 mx-auto mb-4" />
            <p class="text-ink-400 dark:text-ink-400 text-lg">No books yet.</p>
            <p class="text-ink-300 dark:text-ink-500 text-sm mt-1">
              Books will appear here once they are scanned from your library folders.
            </p>
          </div>
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {#each viewBooks as book (book.id)}
            <div
              class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden hover:shadow-md transition-shadow"
            >
              {#if book.cover_image_url}
                <div class="aspect-[2/3] bg-ink-100 dark:bg-ink-800">
                  <img
                    src={book.cover_image_url}
                    alt={book.title}
                    class="w-full h-full object-cover"
                  />
                </div>
              {:else}
                <div class="aspect-[2/3] bg-ink-100 dark:bg-ink-800 flex items-center justify-center">
                  <BookOpen class="w-10 h-10 text-ink-300 dark:text-ink-600" />
                </div>
              {/if}
              <div class="p-3">
                <h3 class="font-medium text-sm text-ink-900 dark:text-cream-100 truncate" title={book.title}>
                  {book.title}
                </h3>
                {#if book.publisher}
                  <p class="text-xs text-ink-400 dark:text-ink-500 truncate mt-0.5">{book.publisher}</p>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {:else if mode === "create" || mode === "edit"}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 animate-fade-in"
    >
      <div class="flex items-center justify-between mb-5">
        <h2 class="text-xl font-display font-bold text-ink-900 dark:text-cream-100">
          {mode === "edit" ? "Edit Library" : "Create Library"}
        </h2>
        <button
          onclick={() => {
            if (mode === "edit" && editId) {
              routerStore.navigate(`libraries/${editId}`);
            } else {
              routerStore.navigate("libraries");
            }
          }}
          class="text-ink-300 hover:text-ink-500 dark:hover:text-ink-200 transition-colors"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      {#if formError}
        <div
          class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm mb-4 animate-scale-in"
        >
          {formError}
        </div>
      {/if}

      <form onsubmit={handleSubmit} class="space-y-5">
        <div>
          <label
            for="lib-name"
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >Name</label
          >
          <input
            id="lib-name"
            type="text"
            bind:value={formName}
            placeholder="e.g. Fiction, Audiobooks"
            class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
            disabled={saving}
          />
        </div>

        <div>
          <span
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
            >Folders</span
          >
          <div class="space-y-2">
            {#each formPaths as entry, i (entry.id)}
              <div class="flex items-center gap-2">
                <FolderOpen
                  class="w-4 h-4 text-ink-300 flex-shrink-0"
                />
                <input
                  type="text"
                  value={entry.value}
                  oninput={(e) => {
                    formPaths[i] = { ...formPaths[i], value: e.currentTarget.value };
                  }}
                  placeholder="/path/to/books"
                  class="flex-1 px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 font-mono text-sm transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
                  disabled={saving}
                />
                {#if formPaths.length > 1}
                  <button
                    type="button"
                    onclick={() => {
                      formPaths = formPaths.filter((_, idx) => idx !== i);
                    }}
                    class="p-2 text-ink-300 hover:text-danger-600 transition-colors"
                    title="Remove folder"
                    disabled={saving}
                  >
                    <X class="w-4 h-4" />
                  </button>
                {/if}
              </div>
            {/each}
          </div>
          <button
            type="button"
            onclick={() => {
              formPaths = [...formPaths, { id: nextPathId++, value: "" }];
            }}
            class="mt-2 inline-flex items-center gap-1.5 text-sm text-accent-600 dark:text-accent-400 hover:text-accent-700 dark:hover:text-accent-300 transition-colors font-medium"
            disabled={saving}
          >
            <Plus class="w-3.5 h-3.5" />
            Add another folder
          </button>
        </div>

        <div>
          <p
            class="text-sm text-ink-400 dark:text-ink-400 mb-2 flex items-center gap-2"
          >
            <FolderOpen class="w-4 h-4" />
            Organization: Book Per Folder
          </p>
        </div>

        <div class="flex items-center gap-3">
          <label class="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              bind:checked={formMonitored}
              class="sr-only peer"
              disabled={saving}
            />
            <div
              class="w-11 h-6 bg-ink-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-accent-500 dark:bg-ink-700 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-ink-200 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-ink-600 peer-checked:bg-accent-600"
            ></div>
          </label>
          <span class="text-sm text-ink-600 dark:text-ink-300"
            >Monitor for new content</span
          >
        </div>

        <div class="flex items-center justify-between pt-2">
          <div class="flex gap-3">
            <button
              type="submit"
              disabled={saving}
              class="px-5 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all text-sm font-semibold disabled:opacity-50 shadow-md shadow-accent-600/20 active:scale-[0.98]"
            >
              {saving
                ? "Saving..."
                : mode === "edit"
                  ? "Update Library"
                  : "Create Library"}
            </button>
            <button
              type="button"
              onclick={() => {
                if (mode === "edit" && editId) {
                  routerStore.navigate(`libraries/${editId}`);
                } else {
                  routerStore.navigate("libraries");
                }
              }}
              disabled={saving}
              class="px-5 py-2.5 border border-ink-200 dark:border-ink-700 text-ink-600 dark:text-ink-300 rounded-xl hover:bg-ink-50 dark:hover:bg-ink-800 transition-all text-sm font-medium"
            >
              Cancel
            </button>
          </div>
          {#if mode === "edit"}
            {#if showDeleteConfirm}
              <div class="flex items-center gap-2 animate-scale-in">
                <span class="text-sm text-danger-600 dark:text-red-400"
                  >Delete this library?</span
                >
                <button
                  type="button"
                  onclick={handleDelete}
                  class="px-3 py-1.5 text-sm bg-danger-600 text-white rounded-lg hover:bg-danger-700 transition-colors"
                >
                  Yes
                </button>
                <button
                  type="button"
                  onclick={() => (showDeleteConfirm = false)}
                  class="px-3 py-1.5 text-sm border border-ink-200 dark:border-ink-700 text-ink-600 dark:text-ink-300 rounded-lg hover:bg-ink-50 dark:hover:bg-ink-800 transition-colors"
                >
                  No
                </button>
              </div>
            {:else}
              <button
                type="button"
                onclick={() => (showDeleteConfirm = true)}
                class="inline-flex items-center gap-1.5 text-sm text-danger-600 hover:text-danger-700 dark:text-red-400 dark:hover:text-red-300 transition-colors"
                disabled={saving}
              >
                <Trash2 class="w-4 h-4" />
                Delete Library
              </button>
            {/if}
          {/if}
        </div>
      </form>
    </div>
  {:else}
    <div class="flex flex-col items-center justify-center py-24 animate-fade-in">
      <LibraryIcon
        class="w-16 h-16 text-ink-200 dark:text-ink-700 mb-6"
      />
      {#if libraryStore.libraries.length === 0}
        <button
          onclick={() => routerStore.navigate("libraries/new")}
          class="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-accent-600 to-accent-700 text-white rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all text-base font-semibold shadow-md shadow-accent-600/20 hover:shadow-lg hover:shadow-accent-600/30 active:scale-[0.98]"
        >
          <Plus class="w-5 h-5" />
          Add A Library
        </button>
      {:else}
        <p class="text-ink-400 dark:text-ink-400">
          Select a library from the sidebar or create a new one.
        </p>
      {/if}
    </div>
  {/if}
</div>
