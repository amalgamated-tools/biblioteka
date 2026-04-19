<script lang="ts">
  import { tick } from "svelte";
  import {
    Tag as TagIcon,
    Plus,
    Check,
    X,
    Pencil,
    Trash2,
  } from "lucide-svelte";
  import { tagStore } from "../stores/tags.svelte";
  import AlertBanner from "./ui/AlertBanner.svelte";
  import Button from "./ui/Button.svelte";
  import TextInput from "./ui/TextInput.svelte";
  import DeleteConfirmation from "./ui/DeleteConfirmation.svelte";

  $effect(() => {
    if (!tagStore.loaded && !tagStore.loading && !tagStore.error) {
      void tagStore.load();
    }
  });

  let showCreateForm = $state(false);
  let newTagName = $state("");
  let creating = $state(false);
  let createError: string | null = $state(null);

  let editingId: string | null = $state(null);
  let editingName = $state("");
  let renaming = $state(false);
  let renameError: string | null = $state(null);
  let renameSuccessMessage = $state("");

  let pendingDeleteId: string | null = $state(null);
  let deleteError: string | null = $state(null);

  async function handleCreate() {
    const name = newTagName.trim();
    if (!name) return;
    creating = true;
    createError = null;
    try {
      await tagStore.add({ name });
      newTagName = "";
      showCreateForm = false;
    } catch (e) {
      createError = e instanceof Error ? e.message : "Failed to create tag";
    } finally {
      creating = false;
    }
  }

  function startEdit(id: string, name: string) {
    editingId = id;
    editingName = name;
    renameError = null;
    renameSuccessMessage = "";
  }

  function cancelEdit() {
    editingId = null;
    editingName = "";
    renameError = null;
    renameSuccessMessage = "";
  }

  async function handleRename() {
    if (!editingId) return;
    const name = editingName.trim();
    if (!name) return;
    renaming = true;
    renameError = null;
    renameSuccessMessage = "";
    try {
      await tagStore.edit(editingId, { name });
      editingId = null;
      editingName = "";
      await tick();
      renameSuccessMessage = `Renamed tag to ${name}.`;
    } catch (e) {
      renameError = e instanceof Error ? e.message : "Failed to rename tag";
    } finally {
      renaming = false;
    }
  }

  function startDelete(id: string) {
    pendingDeleteId = id;
    deleteError = null;
  }

  async function confirmDelete() {
    if (!pendingDeleteId) return;
    deleteError = null;
    try {
      await tagStore.remove(pendingDeleteId);
      pendingDeleteId = null;
    } catch (e) {
      deleteError = e instanceof Error ? e.message : "Failed to delete tag";
      pendingDeleteId = null;
    }
  }

  function cancelDelete() {
    pendingDeleteId = null;
    deleteError = null;
  }

  function cancelCreate() {
    showCreateForm = false;
    newTagName = "";
    createError = null;
  }
</script>

<div>
  <span role="status" class="sr-only">{renameSuccessMessage}</span>
  <span id="tag-rename-error" role="alert" class="sr-only"
    >{renameError ?? ""}</span
  >
  <div class="flex items-center justify-between mb-8">
    <div class="flex items-center gap-3">
      <div
        class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
      >
        <TagIcon
          class="w-5 h-5 text-accent-600 dark:text-accent-400"
          aria-hidden="true"
        />
      </div>
      <h1
        class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
      >
        Tags
      </h1>
    </div>
    {#if !showCreateForm}
      <Button onclick={() => (showCreateForm = true)}>
        <Plus class="w-4 h-4 mr-1" aria-hidden="true" />
        New Tag
      </Button>
    {/if}
  </div>

  {#if tagStore.error}
    <AlertBanner variant="error" class="mb-4">
      {tagStore.error}
      <button
        type="button"
        onclick={() => void tagStore.load()}
        class="underline hover:no-underline font-medium ml-2">Try again</button
      >
    </AlertBanner>
  {/if}

  {#if deleteError}
    <AlertBanner variant="error" class="mb-4">
      <div class="flex items-start justify-between gap-3">
        <span>{deleteError}</span>
        <button
          type="button"
          class="shrink-0 hover:opacity-80 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 rounded-sm"
          aria-label="Dismiss delete error"
          onclick={() => (deleteError = null)}
        >
          <X class="w-4 h-4" aria-hidden="true" />
        </button>
      </div>
    </AlertBanner>
  {/if}

  {#if showCreateForm}
    <div
      class="mb-6 p-5 bg-white dark:bg-ink-900 rounded-2xl border border-ink-200 dark:border-ink-700 shadow-sm"
    >
      <h2 class="text-lg font-semibold text-ink-900 dark:text-cream-100 mb-4">
        Create Tag
      </h2>
      {#if createError}
        <AlertBanner id="create-tag-error" variant="error" class="mb-3"
          >{createError}</AlertBanner
        >
      {/if}
      <div class="space-y-3">
        <p class="text-xs text-ink-500 dark:text-ink-300">
          Fields marked with <span aria-hidden="true">*</span><span
            class="sr-only">an asterisk</span
          > are required.
        </p>
        <div>
          <label
            for="new-tag-name"
            class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
          >
            Name <span class="text-danger-500" aria-hidden="true">*</span>
          </label>
          <TextInput
            id="new-tag-name"
            bind:value={newTagName}
            placeholder="e.g. fiction, sci-fi…"
            disabled={creating}
            aria-required={true}
            aria-invalid={!!createError}
            aria-describedby={createError ? "create-tag-error" : undefined}
          />
        </div>
        <div class="flex gap-2 pt-1">
          <Button
            onclick={handleCreate}
            disabled={creating || !newTagName.trim()}
          >
            <Check class="w-4 h-4 mr-1" aria-hidden="true" />
            {creating ? "Creating…" : "Create"}
          </Button>
          <Button
            variant="secondary"
            onclick={cancelCreate}
            disabled={creating}
          >
            <X class="w-4 h-4 mr-1" aria-hidden="true" />
            Cancel
          </Button>
        </div>
      </div>
    </div>
  {/if}

  {#if tagStore.loading}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800"
    >
      <p role="status" class="text-center text-ink-500 dark:text-ink-300">
        Loading tags...
      </p>
    </div>
  {:else if tagStore.tags.length === 0 && tagStore.loaded && !tagStore.error && !showCreateForm}
    <div
      class="flex flex-col items-center justify-center py-20 text-center"
      aria-live="polite"
    >
      <div
        class="w-16 h-16 bg-ink-100 dark:bg-ink-800 rounded-2xl flex items-center justify-center mb-4"
      >
        <TagIcon
          class="w-8 h-8 text-ink-400 dark:text-ink-500"
          aria-hidden="true"
        />
      </div>
      <h2 class="text-xl font-semibold text-ink-700 dark:text-ink-300 mb-2">
        No tags yet
      </h2>
      <p class="text-ink-500 dark:text-ink-400 mb-6">
        Create tags to organize your books by genre, topic, or any category you
        like.
      </p>
      <Button onclick={() => (showCreateForm = true)}>
        <Plus class="w-4 h-4 mr-1" aria-hidden="true" />
        Create Your First Tag
      </Button>
    </div>
  {:else}
    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden"
    >
      <table class="w-full text-sm" aria-label="Tags">
        <thead>
          <tr
            class="text-left text-ink-500 dark:text-ink-300 border-b border-ink-100 dark:border-ink-800 bg-ink-50/50 dark:bg-ink-800/30"
          >
            <th scope="col" class="px-5 py-3 font-medium">Name</th>
            <th scope="col" class="px-5 py-3 font-medium">Created</th>
            <th scope="col" class="px-5 py-3 font-medium">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {#each tagStore.tags as tag (tag.id)}
            <tr
              class="border-b border-ink-50 dark:border-ink-800 last:border-0 hover:bg-ink-50/50 dark:hover:bg-ink-800/30 transition-colors"
            >
              <td class="px-5 py-3">
                {#if editingId === tag.id}
                  <form
                    class="flex items-center gap-2"
                    onsubmit={(e) => {
                      e.preventDefault();
                      void handleRename();
                    }}
                  >
                    <span class="text-xs text-danger-600 dark:text-danger-400">
                      {renameError ?? ""}
                    </span>
                    <TextInput
                      bind:value={editingName}
                      disabled={renaming}
                      aria-label="Tag name"
                      aria-invalid={!!renameError}
                      aria-describedby={renameError
                        ? "tag-rename-error"
                        : undefined}
                      class="py-1.5 text-sm"
                    />
                    <button
                      type="submit"
                      disabled={renaming || !editingName.trim()}
                      aria-label="Save tag name"
                      class="p-1.5 rounded-lg text-success-600 dark:text-green-400 hover:bg-success-50 dark:hover:bg-green-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <Check class="w-4 h-4" aria-hidden="true" />
                    </button>
                    <button
                      type="button"
                      onclick={cancelEdit}
                      disabled={renaming}
                      aria-label="Cancel rename"
                      class="p-1.5 rounded-lg text-ink-500 hover:bg-ink-100 dark:hover:bg-ink-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <X class="w-4 h-4" aria-hidden="true" />
                    </button>
                  </form>
                {:else}
                  <span class="font-medium text-ink-900 dark:text-cream-100">
                    <span
                      class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-accent-100 text-accent-800 dark:bg-accent-800/30 dark:text-accent-300 mr-2"
                    >
                      {tag.name}
                    </span>
                  </span>
                {/if}
              </td>
              <td class="px-5 py-3 text-ink-500 dark:text-ink-400">
                {new Date(tag.created_at).toLocaleDateString()}
              </td>
              <td class="px-5 py-3 text-right">
                {#if pendingDeleteId === tag.id}
                  <DeleteConfirmation
                    itemId={tag.id}
                    itemName={tag.name}
                    onConfirm={confirmDelete}
                    onCancel={cancelDelete}
                  />
                {:else if editingId !== tag.id}
                  <div class="flex items-center justify-end gap-1">
                    <button
                      onclick={() => startEdit(tag.id, tag.name)}
                      aria-label={`Rename tag ${tag.name}`}
                      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-ink-500 hover:bg-ink-100 dark:text-ink-400 dark:hover:bg-ink-700/50 transition-colors"
                    >
                      <Pencil class="w-3.5 h-3.5" aria-hidden="true" />
                      Rename
                    </button>
                    <button
                      onclick={() => startDelete(tag.id)}
                      aria-label={`Delete tag ${tag.name}`}
                      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-danger-600 hover:bg-danger-50 dark:text-red-400 dark:hover:bg-danger-700/10 transition-colors"
                    >
                      <Trash2 class="w-3.5 h-3.5" aria-hidden="true" />
                      Delete
                    </button>
                  </div>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
