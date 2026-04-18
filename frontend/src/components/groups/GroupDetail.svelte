<script lang="ts">
  import {
    ArrowLeft,
    Pencil,
    Trash2,
    Users,
    BookMarked,
    UserPlus,
    UserMinus,
    X,
    Check,
    Share2,
  } from "lucide-svelte";
  import { routerStore } from "../../stores/router.svelte";
  import { groupStore } from "../../stores/groups.svelte";
  import { readingListStore } from "../../stores/reading-lists.svelte";
  import { authStore } from "../../stores/auth.svelte";
  import {
    listGroupMembers,
    addGroupMember,
    removeGroupMember,
    listGroupReadingLists,
    shareListWithGroup,
    unshareListFromGroup,
  } from "../../lib/api";
  import { autofocusFirstButton } from "../../lib/actions";
  import type {
    ReadingGroup,
    ReadingGroupMember,
    ReadingList,
  } from "../../types";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    groupId: string;
  }

  let { groupId }: Props = $props();

  let group = $state<ReadingGroup | null>(null);
  let error: string | null = $state(null);
  let members: ReadingGroupMember[] = $state.raw([]);
  let membersLoading = $state(false);
  let membersError: string | null = $state(null);
  let sharedLists: ReadingList[] = $state.raw([]);
  let sharedListsLoading = $state(false);
  let sharedListsError: string | null = $state(null);

  // Edit group
  let editing = $state(false);
  let editName = $state("");
  let editDescription = $state("");
  let saving = $state(false);
  let saveError: string | null = $state(null);
  let editNameInvalid = $derived(!!saveError);

  // Delete group
  let confirmDelete = $state(false);
  let deleting = $state(false);

  // Add member
  let showAddMember = $state(false);
  let newMemberUserId = $state("");
  let addingMember = $state(false);
  let addMemberError: string | null = $state(null);

  // Remove member confirm
  let confirmRemoveMemberId: string | null = $state(null);
  let removingMemberId: string | null = $state(null);

  // Share list
  let shareListId = $state("");
  let sharingList = $state(false);
  let shareListError: string | null = $state(null);
  let confirmUnshareListId: string | null = $state(null);
  let unsharingListId: string | null = $state(null);

  let currentUserId = $derived(authStore.user?.id ?? "");
  let isOwner = $derived(group !== null && group.owner_id === currentUserId);

  $effect(() => {
    if (!groupStore.loaded && !groupStore.loading) {
      void groupStore.load();
    }
  });

  $effect(() => {
    const found = groupStore.groups.find((g) => g.id === groupId) ?? null;
    group = found;
    error =
      !found && groupStore.loaded
        ? (groupStore.loadError ?? "Group not found.")
        : null;
  });

  $effect(() => {
    if (!groupId) return;
    let cancelled = false;

    // Reset all transient UI state so previous-group flows can't bleed into the new group.
    editing = false;
    editName = "";
    editDescription = "";
    saving = false;
    saveError = null;
    confirmDelete = false;
    deleting = false;
    showAddMember = false;
    newMemberUserId = "";
    addingMember = false;
    addMemberError = null;
    confirmRemoveMemberId = null;
    removingMemberId = null;
    shareListId = "";
    sharingList = false;
    shareListError = null;
    confirmUnshareListId = null;
    unsharingListId = null;
    members = [];
    sharedLists = [];

    membersLoading = true;
    membersError = null;
    listGroupMembers(groupId)
      .then((fetched) => {
        if (!cancelled) members = fetched;
      })
      .catch((e) => {
        if (!cancelled)
          membersError =
            e instanceof Error ? e.message : "Failed to load members";
      })
      .finally(() => {
        if (!cancelled) membersLoading = false;
      });

    sharedListsLoading = true;
    sharedListsError = null;
    listGroupReadingLists(groupId)
      .then((fetched) => {
        if (!cancelled) sharedLists = fetched;
      })
      .catch((e) => {
        if (!cancelled)
          sharedListsError =
            e instanceof Error ? e.message : "Failed to load shared lists";
      })
      .finally(() => {
        if (!cancelled) sharedListsLoading = false;
      });

    return () => {
      cancelled = true;
    };
  });

  $effect(() => {
    if (!readingListStore.loaded && !readingListStore.loading) {
      void readingListStore.load();
    }
  });

  async function loadMembers(): Promise<ReadingGroupMember[] | null> {
    membersLoading = true;
    membersError = null;
    try {
      const fetched = await listGroupMembers(groupId);
      members = fetched;
      return fetched;
    } catch (e) {
      membersError = e instanceof Error ? e.message : "Failed to load members";
      return null;
    } finally {
      membersLoading = false;
    }
  }

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

  function startEditing() {
    if (!group) return;
    editName = group.name;
    editDescription = group.description ?? "";
    editing = true;
    saveError = null;
  }

  function cancelEditing() {
    editing = false;
    saveError = null;
  }

  async function saveEdit() {
    if (!group || !editName.trim()) return;
    saving = true;
    saveError = null;
    try {
      await groupStore.update(group.id, {
        name: editName.trim(),
        description: editDescription.trim() || null,
      });
      editing = false;
    } catch (e) {
      saveError = e instanceof Error ? e.message : "Failed to update group";
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!group) return;
    deleting = true;
    try {
      await groupStore.remove(group.id);
      routerStore.navigate("groups");
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to delete group";
      confirmDelete = false;
    } finally {
      deleting = false;
    }
  }

  async function handleAddMember() {
    const uid = newMemberUserId.trim();
    if (!uid) return;
    addingMember = true;
    addMemberError = null;
    try {
      await addGroupMember(groupId, uid);
      newMemberUserId = "";
      showAddMember = false;
      const fetched = await loadMembers();
      if (fetched !== null) {
        groupStore.setMemberCount(groupId, fetched.length);
      }
    } catch (e) {
      addMemberError = e instanceof Error ? e.message : "Failed to add member";
    } finally {
      addingMember = false;
    }
  }

  async function handleRemoveMember(userId: string) {
    removingMemberId = userId;
    try {
      await removeGroupMember(groupId, userId);
      confirmRemoveMemberId = null;
      const fetched = await loadMembers();
      if (fetched !== null) {
        groupStore.setMemberCount(groupId, fetched.length);
      }
    } catch (e) {
      membersError = e instanceof Error ? e.message : "Failed to remove member";
    } finally {
      removingMemberId = null;
    }
  }

  // Lists available to share (owned by current user, not already shared)
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

<div>
  <div class="mb-6">
    <a
      href="#groups"
      class="inline-flex items-center gap-1.5 text-sm text-ink-500 dark:text-ink-400 hover:text-ink-700 dark:hover:text-ink-200 transition-colors mb-4"
    >
      <ArrowLeft class="w-4 h-4" aria-hidden="true" />
      All Groups
    </a>
  </div>

  {#if error}
    <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
  {/if}

  {#if group}
    <!-- Header -->
    <div class="mb-8">
      {#if editing}
        <div
          class="p-5 bg-white dark:bg-ink-900 rounded-2xl border border-ink-200 dark:border-ink-700 shadow-sm mb-6"
        >
          <h2
            class="text-lg font-semibold text-ink-900 dark:text-cream-100 mb-4"
          >
            Edit Group
          </h2>
          {#if saveError}
            <AlertBanner id="edit-group-error" variant="error" class="mb-3"
              >{saveError}</AlertBanner
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
                for="edit-group-name"
                class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
              >
                Name <span class="text-danger-500" aria-hidden="true">*</span>
              </label>
              <TextInput
                id="edit-group-name"
                bind:value={editName}
                disabled={saving}
                aria-required={true}
                aria-invalid={editNameInvalid}
                aria-describedby={editNameInvalid
                  ? "edit-group-error"
                  : undefined}
              />
            </div>
            <div>
              <label
                for="edit-group-desc"
                class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
              >
                Description
              </label>
              <TextInput
                id="edit-group-desc"
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
              <Users
                class="w-6 h-6 text-accent-600 dark:text-accent-400"
                aria-hidden="true"
              />
            </div>
            <div class="min-w-0">
              <h1
                class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
              >
                {group.name}
              </h1>
              <p class="text-sm text-ink-500 dark:text-ink-400 mt-0.5">
                {group.member_count}
                {group.member_count === 1 ? "member" : "members"}
              </p>
            </div>
          </div>
          {#if isOwner}
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
                <div
                  class="flex items-center gap-2"
                  role="group"
                  aria-labelledby={`delete-group-prompt-${groupId}`}
                  use:autofocusFirstButton
                >
                  <span
                    id={`delete-group-prompt-${groupId}`}
                    class="text-sm text-ink-600 dark:text-ink-300 mr-1"
                    >Delete this group?</span
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
          {/if}
        </div>

        {#if group.description}
          <p class="text-ink-600 dark:text-ink-300 mb-6">
            {group.description}
          </p>
        {/if}
      {/if}
    </div>

    <!-- Members section -->
    <section aria-labelledby="group-members-heading" class="mb-8">
      <div class="flex items-center justify-between mb-4">
        <h2
          id="group-members-heading"
          class="text-lg font-semibold text-ink-900 dark:text-cream-100"
        >
          Members
        </h2>
        {#if isOwner && !showAddMember}
          <Button variant="secondary" onclick={() => (showAddMember = true)}>
            <UserPlus class="w-4 h-4 mr-1" aria-hidden="true" />
            Add Member
          </Button>
        {/if}
      </div>

      {#if showAddMember}
        <div
          class="mb-4 p-4 bg-white dark:bg-ink-900 rounded-xl border border-ink-200 dark:border-ink-700"
        >
          <h3
            class="text-sm font-semibold text-ink-900 dark:text-cream-100 mb-3"
          >
            Add Member by User ID
          </h3>
          {#if addMemberError}
            <AlertBanner id="add-member-error" variant="error" class="mb-3"
              >{addMemberError}</AlertBanner
            >
          {/if}
          <div class="flex gap-2">
            <TextInput
              id="new-member-user-id"
              bind:value={newMemberUserId}
              placeholder="User ID"
              disabled={addingMember}
              aria-label="User ID to add"
              aria-describedby={addMemberError ? "add-member-error" : undefined}
            />
            <Button
              onclick={handleAddMember}
              disabled={addingMember || !newMemberUserId.trim()}
            >
              <Check class="w-4 h-4 mr-1" aria-hidden="true" />
              {addingMember ? "Adding…" : "Add"}
            </Button>
            <Button
              variant="secondary"
              onclick={() => {
                showAddMember = false;
                newMemberUserId = "";
                addMemberError = null;
              }}
              disabled={addingMember}
            >
              <X class="w-4 h-4 mr-1" aria-hidden="true" />
              Cancel
            </Button>
          </div>
        </div>
      {/if}

      {#if membersError}
        <AlertBanner variant="error" class="mb-3">{membersError}</AlertBanner>
      {/if}

      <ul
        class="divide-y divide-ink-100 dark:divide-ink-800 bg-white dark:bg-ink-900 rounded-xl border border-ink-200 dark:border-ink-700"
        role="list"
      >
        {#each members as member (member.user_id)}
          <li class="flex items-center justify-between px-4 py-3 gap-3">
            <div class="flex items-center gap-3 min-w-0">
              <div
                class="w-8 h-8 bg-accent-100 dark:bg-accent-800/30 rounded-full flex items-center justify-center flex-shrink-0"
              >
                <span
                  class="text-xs font-semibold text-accent-700 dark:text-accent-300"
                  aria-hidden="true"
                >
                  {member.user_name.slice(0, 1).toUpperCase()}
                </span>
              </div>
              <div class="min-w-0">
                <p
                  class="text-sm font-medium text-ink-900 dark:text-cream-100 truncate"
                >
                  {member.user_name}
                </p>
                <p class="text-xs text-ink-500 dark:text-ink-400 capitalize">
                  {member.role}
                </p>
              </div>
            </div>
            {#if isOwner && member.user_id !== currentUserId}
              {#if confirmRemoveMemberId === member.user_id}
                <div
                  class="flex items-center gap-2 flex-shrink-0"
                  role="group"
                  aria-labelledby={`remove-member-prompt-${member.user_id}`}
                  use:autofocusFirstButton
                >
                  <span
                    id={`remove-member-prompt-${member.user_id}`}
                    class="text-xs text-ink-600 dark:text-ink-300">Remove?</span
                  >
                  <Button
                    variant="danger"
                    onclick={() => handleRemoveMember(member.user_id)}
                    disabled={removingMemberId === member.user_id}
                  >
                    {removingMemberId === member.user_id ? "Removing…" : "Yes"}
                  </Button>
                  <Button
                    variant="secondary"
                    onclick={() => (confirmRemoveMemberId = null)}
                    disabled={removingMemberId === member.user_id}
                  >
                    No
                  </Button>
                </div>
              {:else}
                <button
                  class="p-1.5 text-ink-400 hover:text-danger-500 dark:hover:text-danger-400 transition-colors rounded"
                  onclick={() => (confirmRemoveMemberId = member.user_id)}
                  aria-label={`Remove ${member.user_name} from group`}
                >
                  <UserMinus class="w-4 h-4" aria-hidden="true" />
                </button>
              {/if}
            {/if}
          </li>
        {/each}
        {#if !membersLoading && members.length === 0}
          <li
            class="px-4 py-6 text-center text-sm text-ink-500 dark:text-ink-400"
          >
            No members yet.
          </li>
        {/if}
      </ul>
    </section>

    <!-- Shared reading lists section -->
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
        <AlertBanner variant="error" class="mb-3"
          >{sharedListsError}</AlertBanner
        >
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
          <Button
            onclick={handleShareList}
            disabled={sharingList || !shareListId}
          >
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
                    class="text-xs text-ink-600 dark:text-ink-300"
                    >Unshare?</span
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
          <li
            class="px-4 py-6 text-center text-sm text-ink-500 dark:text-ink-400"
          >
            No reading lists shared with this group yet.
          </li>
        {/if}
      </ul>
    </section>
  {/if}
</div>
