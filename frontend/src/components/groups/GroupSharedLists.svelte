<script lang="ts">
  import { BookMarked, X, Share2 } from "lucide-svelte";
  import { readingListStore } from "../../stores/reading-lists.svelte";
  import {
    listGroupReadingLists,
    shareListWithGroup,
    unshareListFromGroup,
  } from "../../lib/api";
  import { autofocusFirstButton } from "../../lib/actions";
  import type { ReadingList } from "../../types";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";

  interface Props {
    groupId: string;
    isOwner: boolean;
  }

  let { groupId, isOwner }: Props = $props();

  let sharedLists: ReadingList[] = $state.raw([]);
  let sharedListsLoading = $state(false);
  let sharedListsError: string | null = $state(null);

  let shareListId = $state("");
  let sharingList = $state(false);
  let shareListError: string | null = $state(null);
  let confirmUnshareListId: string | null = $state(null);
  let unsharingListId: string | null = $state(null);

  $effect(() => {
    if (!readingListStore.loaded && !readingListStore.loading) {
      void readingListStore.load();
    }
  });

  $effect(() => {
    if (!groupId) return;
    let cancelled = false;
    shareListId = "";
    sharingList = false;
    shareListError = null;
    confirmUnshareListId = null;
    unsharingListId = null;
    sharedLists = [];

    sharedListsLoading = true;
    sharedListsError = null;
    listGroupReadingLists(groupId)
      .then((fetched) => {
        if (!cancelled) {
          sharedLists = fetched;
          sharedListsLoading = false;
        }
      })
      .catch((e) => {
        if (!cancelled) {
          sharedListsError =
            e instanceof Error ? e.message : "Failed to load shared lists";
          sharedListsLoading = false;
        }
      });

    return () => {
      cancelled = true;
    };
  });

  async function loadSharedLists() {
    sharedListsLoading = true;
    sharedListsError = null;
    try {
      sharedLists = await listGroupReadingLists(groupId);
    } catch (e) {
      sharedListsError =
        e instanceof Error ? e.message : "Failed to load shared lists";
    } finally {
      sharedListsLoading = false;
    }
  }

  let availableListsToShare = $derived(
    readingListStore.lists.filter(
      (l) => !sharedLists.some((sl) => sl.id === l.id),
    ),
  );

  async function handleShareList() {
    if (!shareListId) return;
    sharingList = true;
    shareListError = null;
    try {
      await shareListWithGroup(groupId, shareListId);
      shareListId = "";
      await loadSharedLists();
    } catch (e) {
      shareListError = e instanceof Error ? e.message : "Failed to share list";
    } finally {
      sharingList = false;
    }
  }

  async function handleUnshareList(listId: string) {
    unsharingListId = listId;
    try {
      await unshareListFromGroup(groupId, listId);
      confirmUnshareListId = null;
      await loadSharedLists();
    } catch (e) {
      sharedListsError =
        e instanceof Error ? e.message : "Failed to unshare list";
    } finally {
      unsharingListId = null;
    }
  }
</script>

<section aria-labelledby="group-lists-heading" class="mb-8">
  <div class="flex items-center justify-between mb-4">
    <h2
      id="group-lists-heading"
      class="text-lg font-semibold text-ink-900 dark:text-cream-100"
    >
      Shared Reading Lists
    </h2>
  </div>

  {#if sharedListsError}
    <AlertBanner variant="error" class="mb-3">{sharedListsError}</AlertBanner>
  {/if}

  {#if shareListError}
    <AlertBanner variant="error" class="mb-3">{shareListError}</AlertBanner>
  {/if}

  {#if availableListsToShare.length > 0}
    <div class="flex gap-2 mb-4">
      <div class="relative flex-1 max-w-xs">
        <label for="share-list-select" class="sr-only"
          >Select a reading list to share</label
        >
        <select
          id="share-list-select"
          bind:value={shareListId}
          class="w-full rounded-xl border border-ink-200 dark:border-ink-700 bg-white dark:bg-ink-900 text-ink-900 dark:text-cream-100 text-sm px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent-400"
        >
          <option value="">Share a reading list…</option>
          {#each availableListsToShare as list (list.id)}
            <option value={list.id}>{list.name}</option>
          {/each}
        </select>
      </div>
      <Button onclick={handleShareList} disabled={sharingList || !shareListId}>
        <Share2 class="w-4 h-4 mr-1" aria-hidden="true" />
        {sharingList ? "Sharing…" : "Share"}
      </Button>
    </div>
  {/if}

  <ul
    class="divide-y divide-ink-100 dark:divide-ink-800 bg-white dark:bg-ink-900 rounded-xl border border-ink-200 dark:border-ink-700"
    role="list"
  >
    {#each sharedLists as list (list.id)}
      <li class="flex items-center justify-between px-4 py-3 gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div
            class="w-8 h-8 bg-accent-100 dark:bg-accent-800/30 rounded-lg flex items-center justify-center flex-shrink-0"
          >
            <BookMarked
              class="w-4 h-4 text-accent-600 dark:text-accent-400"
              aria-hidden="true"
            />
          </div>
          <div class="min-w-0">
            <a
              href={`#reading-lists/${list.id}`}
              class="text-sm font-medium text-ink-900 dark:text-cream-100 hover:text-accent-600 dark:hover:text-accent-400 transition-colors truncate block"
            >
              {list.name}
            </a>
            <p class="text-xs text-ink-500 dark:text-ink-400 mt-0.5">
              {list.book_count}
              {list.book_count === 1 ? "book" : "books"}
            </p>
          </div>
        </div>
        {#if isOwner || readingListStore.lists.some((rl) => rl.id === list.id)}
          {#if confirmUnshareListId === list.id}
            <div
              class="flex items-center gap-2 flex-shrink-0"
              role="group"
              aria-labelledby={`unshare-list-prompt-${list.id}`}
              use:autofocusFirstButton
            >
              <span
                id={`unshare-list-prompt-${list.id}`}
                class="text-xs text-ink-600 dark:text-ink-300">Unshare?</span
              >
              <Button
                variant="danger"
                onclick={() => handleUnshareList(list.id)}
                disabled={unsharingListId === list.id}
              >
                {unsharingListId === list.id ? "Removing…" : "Yes"}
              </Button>
              <Button
                variant="secondary"
                onclick={() => (confirmUnshareListId = null)}
                disabled={unsharingListId === list.id}
              >
                No
              </Button>
            </div>
          {:else}
            <button
              class="p-1.5 text-ink-400 hover:text-danger-500 dark:hover:text-danger-400 transition-colors rounded"
              onclick={() => (confirmUnshareListId = list.id)}
              aria-label={`Unshare ${list.name} from group`}
            >
              <X class="w-4 h-4" aria-hidden="true" />
            </button>
          {/if}
        {/if}
      </li>
    {/each}
    {#if !sharedListsLoading && sharedLists.length === 0}
      <li class="px-4 py-6 text-center text-sm text-ink-500 dark:text-ink-400">
        No reading lists shared with this group yet.
      </li>
    {/if}
  </ul>
</section>
