<script lang="ts">
  import { libraryStore } from "../../stores/libraries.svelte";
  import { routerStore } from "../../stores/router.svelte";
  import {
    Plus,
    FolderOpen,
    Trash2,
    X,
  } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";

  interface Props {
    mode: "create" | "edit";
    editId: string;
  }

  let { mode, editId }: Props = $props();

  let saving = $state(false);
  let editingId: string | null = $state(null);
  let formName = $state("");
  let nextPathId = 0;
  let formPaths: { id: number; value: string }[] = $state([
    { id: nextPathId++, value: "" },
  ]);
  let formMonitored = $state(false);
  let formError: string | null = $state(null);
  let showDeleteConfirm = $state(false);

  // React to mode/editId changes to populate form
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
      } else {
        editingId = null;
        formName = "";
        formPaths = [{ id: nextPathId++, value: "" }];
        formMonitored = false;
        formError = "Library not found";
      }
    }
  });

  function navigateBack() {
    if (mode === "edit" && editId) {
      routerStore.navigate(`libraries/${editId}`);
    } else {
      routerStore.navigate("libraries");
    }
  }

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
    saving = true;
    try {
      await libraryStore.remove(editingId);
      routerStore.navigate("libraries");
    } catch (e) {
      formError = e instanceof Error ? e.message : "Failed to delete library";
      showDeleteConfirm = false;
    } finally {
      saving = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 animate-fade-in"
>
  <div class="flex items-center justify-between mb-5">
    <h2 class="text-xl font-display font-bold text-ink-900 dark:text-cream-100">
      {mode === "edit" ? "Edit Library" : "Create Library"}
    </h2>
    <button
      onclick={navigateBack}
      class="text-ink-300 hover:text-ink-500 dark:hover:text-ink-200 transition-colors"
      aria-label="Close form"
    >
      <X class="w-5 h-5" />
    </button>
  </div>

  {#if formError}
    <AlertBanner variant="error" class="mb-4">{formError}</AlertBanner>
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
                aria-label="Remove folder"
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
          onclick={navigateBack}
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
              disabled={saving}
              class="px-3 py-1.5 text-sm bg-danger-600 text-white rounded-lg hover:bg-danger-700 transition-colors disabled:opacity-50"
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
