<script lang="ts">
  import {
    ArrowLeft,
    Pencil,
    Trash2,
    BookMarked,
    X,
    Check,
  } from "lucide-svelte";
  import { routerStore } from "../../stores/router.svelte";
  import { readingListStore } from "../../stores/reading-lists.svelte";
  import { listReadingListBooks } from "../../lib/api";
  import { autofocusFirstButton } from "../../lib/actions";
  import type { ReadingList } from "../../types";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import BookList from "../ui/BookList.svelte";

  interface Props {
    listId: string;
  }

  let { listId }: Props = $props();

  let list: ReadingList | null = $state(null);
  let error: string | null = $state(null);
  let editing = $state(false);
  let editName = $state("");
  let editDescription = $state("");
  let saving = $state(false);
  let saveError: string | null = $state(null);
  let confirmDelete = $state(false);
  let deleting = $state(false);

  // Load list from store or server.
  $effect(() => {
    const nextList = readingListStore.lists.find((l) => l.id === listId) ?? null;
    list = nextList;
    error =
      !nextList && readingListStore.loaded
        ? (readingListStore.loadError ?? "Reading list not found.")
        : null;
  });

  $effect(() => {
    if (!readingListStore.loaded && !readingListStore.loading) {
      void readingListStore.load();
    }
  });

  function startEditing() {
    if (!list) return;
    editName = list.name;
    editDescription = list.description ?? "";
    editing = true;
    saveError = null;
  }

  function cancelEditing() {
    editing = false;
    saveError = null;
  }

  async function saveEdit() {
    if (!list || !editName.trim()) return;
    saving = true;
    saveError = null;
    try {
      await readingListStore.update(list.id, {
        name: editName.trim(),
        description: editDescription.trim() || null,
      });
      editing = false;
    } catch (e) {
      saveError =
        e instanceof Error ? e.message : "Failed to update reading list";
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!list) return;
    deleting = true;
    try {
      await readingListStore.remove(list.id);
      routerStore.navigate("reading-lists");
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to delete reading list";
      confirmDelete = false;
    } finally {
      deleting = false;
    }
  }

  function fetchBooks(limit: number, offset: number) {
    return listReadingListBooks(listId, limit, offset);
  }
</script>

<div>
  <div class="mb-6">
    <a
      href="#reading-lists"
      class="inline-flex items-center gap-1.5 text-sm text-ink-500 dark:text-ink-400 hover:text-ink-700 dark:hover:text-ink-200 transition-colors mb-4"
    >
      <ArrowLeft class="w-4 h-4" aria-hidden="true" />
      All Reading Lists
    </a>
  </div>

  {#if error}
    <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
  {/if}

  {#if list}
    <div class="mb-8">
      {#if editing}
        <div
          class="p-5 bg-white dark:bg-ink-900 rounded-2xl border border-ink-200 dark:border-ink-700 shadow-sm mb-6"
        >
          <h2
            class="text-lg font-semibold text-ink-900 dark:text-cream-100 mb-4"
          >
            Edit Reading List
          </h2>
          {#if saveError}
            <AlertBanner variant="error" class="mb-3">{saveError}</AlertBanner>
          {/if}
          <div class="space-y-3">
            <div>
              <label
                for="edit-list-name"
                class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
              >
                Name <span class="text-danger-500" aria-hidden="true">*</span>
              </label>
              <TextInput
                id="edit-list-name"
                bind:value={editName}
                disabled={saving}
              />
            </div>
            <div>
              <label
                for="edit-list-desc"
                class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
              >
                Description
              </label>
              <TextInput
                id="edit-list-desc"
                bind:value={editDescription}
                placeholder="Optional description"
                disabled={saving}
              />
            </div>
            <div class="flex gap-2 pt-1">
              <Button onclick={saveEdit} disabled={saving || !editName.trim()}>
                <Check class="w-4 h-4 mr-1" aria-hidden="true" />
                {saving ? "Saving…" : "Save"}
              </Button>
              <Button
                variant="secondary"
                onclick={cancelEditing}
                disabled={saving}
              >
                <X class="w-4 h-4 mr-1" aria-hidden="true" />
                Cancel
              </Button>
            </div>
          </div>
        </div>
      {:else}
        <div class="flex items-start justify-between gap-4 mb-6">
          <div class="flex items-center gap-3 min-w-0">
            <div
              class="w-12 h-12 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center flex-shrink-0"
            >
              <BookMarked
                class="w-6 h-6 text-accent-600 dark:text-accent-400"
                aria-hidden="true"
              />
            </div>
            <div class="min-w-0">
              <h1
                class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
              >
                {list.name}
              </h1>
              <p class="text-sm text-ink-500 dark:text-ink-400 mt-0.5">
                {list.book_count}
                {list.book_count === 1 ? "book" : "books"}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <Button variant="secondary" onclick={startEditing}>
              <Pencil class="w-4 h-4 mr-1" aria-hidden="true" />
              Edit
            </Button>
            {#if !confirmDelete}
              <Button variant="danger" onclick={() => (confirmDelete = true)}>
                <Trash2 class="w-4 h-4 mr-1" aria-hidden="true" />
                Delete
              </Button>
            {:else}
              <div class="flex items-center gap-2" use:autofocusFirstButton>
                <span class="text-sm text-ink-600 dark:text-ink-300 mr-1"
                  >Delete this list?</span
                >
                <Button
                  variant="danger"
                  onclick={handleDelete}
                  disabled={deleting}
                >
                  {deleting ? "Deleting…" : "Yes, delete"}
                </Button>
                <Button
                  variant="secondary"
                  onclick={() => (confirmDelete = false)}
                  disabled={deleting}
                >
                  Cancel
                </Button>
              </div>
            {/if}
          </div>
        </div>

        {#if list.description}
          <p class="text-ink-600 dark:text-ink-300 mb-6">{list.description}</p>
        {/if}
      {/if}
    </div>

    <BookList {fetchBooks} pageSize={24} />
  {/if}
</div>
