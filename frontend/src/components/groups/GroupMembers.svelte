<script lang="ts">
  import { UserPlus, UserMinus, X, Check } from "lucide-svelte";
  import { groupStore } from "../../stores/groups.svelte";
  import {
    listGroupMembers,
    addGroupMember,
    removeGroupMember,
  } from "../../lib/api";
  import { autofocusFirstButton } from "../../lib/actions";
  import type { ReadingGroupMember } from "../../types";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    groupId: string;
    isOwner: boolean;
    currentUserId: string;
  }

  let { groupId, isOwner, currentUserId }: Props = $props();

  let members: ReadingGroupMember[] = $state.raw([]);
  let membersLoading = $state(false);
  let membersError: string | null = $state(null);

  let showAddMember = $state(false);
  let newMemberUserId = $state("");
  let addingMember = $state(false);
  let addMemberError: string | null = $state(null);

  let confirmRemoveMemberId: string | null = $state(null);
  let removingMemberId: string | null = $state(null);

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

  $effect(() => {
    if (!groupId) return;
    showAddMember = false;
    newMemberUserId = "";
    addingMember = false;
    addMemberError = null;
    confirmRemoveMemberId = null;
    removingMemberId = null;
    members = [];
    void loadMembers();
  });

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
</script>

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
      <h3 class="text-sm font-semibold text-ink-900 dark:text-cream-100 mb-3">
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
      <li class="px-4 py-6 text-center text-sm text-ink-500 dark:text-ink-400">
        No members yet.
      </li>
    {/if}
  </ul>
</section>
