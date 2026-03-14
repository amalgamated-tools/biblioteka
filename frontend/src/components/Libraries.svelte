<script lang="ts">
  import { onMount } from "svelte";
  import { libraryStore } from "../stores/libraries.svelte";
  import { routerStore } from "../stores/router.svelte";
  import {
    Plus,
    FolderOpen,
    Trash2,
    X,
    Library as LibraryIcon,
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

  // Determine mode from subPath: "new", "edit/{id}", or empty
  let mode: "create" | "edit" | "empty" = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp === "new") return "create";
    if (sp.startsWith("edit/")) return "edit";
    return "empty";
  });

  let editId: string = $derived.by(() => {
    const sp = routerStore.subPath;
    if (sp.startsWith("edit/")) return sp.slice(5);
    return "";
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
    }
  });

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
      } else {
        await libraryStore.add(input);
      }
      routerStore.navigate("libraries");
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
      class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm mb-4"
    >
      {error}
    </div>
  {/if}

  {#if mode === "create" || mode === "edit"}
    <div
      class="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 p-6"
    >
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-white">
          {mode === "edit" ? "Edit Library" : "Create Library"}
        </h2>
        <button
          onclick={() => routerStore.navigate("libraries")}
          class="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      {#if formError}
        <div
          class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm mb-4"
        >
          {formError}
        </div>
      {/if}

      <form onsubmit={handleSubmit} class="space-y-4">
        <div>
          <label
            for="lib-name"
            class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
            >Name</label
          >
          <input
            id="lib-name"
            type="text"
            bind:value={formName}
            placeholder="e.g. Fiction, Audiobooks"
            class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
            disabled={saving}
          />
        </div>

        <div>
          <span
            class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
            >Folders</span
          >
          <div class="space-y-2">
            {#each formPaths as entry, i (entry.id)}
              <div class="flex items-center gap-2">
                <FolderOpen
                  class="w-4 h-4 text-slate-400 flex-shrink-0"
                />
                <input
                  type="text"
                  value={entry.value}
                  oninput={(e) => {
                    formPaths[i] = { ...formPaths[i], value: e.currentTarget.value };
                  }}
                  placeholder="/path/to/books"
                  class="flex-1 px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100 font-mono text-sm"
                  disabled={saving}
                />
                {#if formPaths.length > 1}
                  <button
                    type="button"
                    onclick={() => {
                      formPaths = formPaths.filter((_, idx) => idx !== i);
                    }}
                    class="p-2 text-slate-400 hover:text-red-500 transition-colors"
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
            class="mt-2 inline-flex items-center gap-1.5 text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
            disabled={saving}
          >
            <Plus class="w-3.5 h-3.5" />
            Add another folder
          </button>
        </div>

        <div>
          <p
            class="text-sm text-slate-500 dark:text-slate-400 mb-2 flex items-center gap-2"
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
              class="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 dark:bg-slate-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-500 peer-checked:bg-blue-600"
            ></div>
          </label>
          <span class="text-sm text-slate-700 dark:text-slate-300"
            >Monitor for new content</span
          >
        </div>

        <div class="flex items-center justify-between pt-2">
          <div class="flex gap-3">
            <button
              type="submit"
              disabled={saving}
              class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium disabled:opacity-50"
            >
              {saving
                ? "Saving..."
                : mode === "edit"
                  ? "Update Library"
                  : "Create Library"}
            </button>
            <button
              type="button"
              onclick={() => routerStore.navigate("libraries")}
              disabled={saving}
              class="px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors text-sm font-medium"
            >
              Cancel
            </button>
          </div>
          {#if mode === "edit"}
            {#if showDeleteConfirm}
              <div class="flex items-center gap-2">
                <span class="text-sm text-red-600 dark:text-red-400"
                  >Delete this library?</span
                >
                <button
                  type="button"
                  onclick={handleDelete}
                  class="px-3 py-1.5 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
                >
                  Yes
                </button>
                <button
                  type="button"
                  onclick={() => (showDeleteConfirm = false)}
                  class="px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                >
                  No
                </button>
              </div>
            {:else}
              <button
                type="button"
                onclick={() => (showDeleteConfirm = true)}
                class="inline-flex items-center gap-1.5 text-sm text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 transition-colors"
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
    <div class="flex flex-col items-center justify-center py-24">
      <LibraryIcon
        class="w-16 h-16 text-slate-300 dark:text-slate-600 mb-6"
      />
      {#if libraryStore.libraries.length === 0}
        <button
          onclick={() => routerStore.navigate("libraries/new")}
          class="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-base font-medium"
        >
          <Plus class="w-5 h-5" />
          Add A Library
        </button>
      {:else}
        <p class="text-slate-500 dark:text-slate-400">
          Select a library from the sidebar or create a new one.
        </p>
      {/if}
    </div>
  {/if}
</div>
