<script lang="ts">
  import { ArrowLeft } from "lucide-svelte";
  import { groupStore } from "../../stores/groups.svelte";
  import { authStore } from "../../stores/auth.svelte";
  import type { ReadingGroup } from "../../types";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import GroupEditHeader from "./GroupEditHeader.svelte";
  import GroupMembers from "./GroupMembers.svelte";
  import GroupSharedLists from "./GroupSharedLists.svelte";

  interface Props {
    groupId: string;
  }

  let { groupId }: Props = $props();

  let group = $state<ReadingGroup | null>(null);
  let error: string | null = $state(null);

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
    <GroupEditHeader
      {group}
      {groupId}
      {isOwner}
      onDeleteError={(message) => (error = message)}
    />
    <GroupMembers {groupId} {isOwner} {currentUserId} />
    <GroupSharedLists {groupId} {isOwner} />
  {/if}
</div>
