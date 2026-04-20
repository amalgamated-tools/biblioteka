<script lang="ts">
  import { Pencil, Trash2, Users, X, Check } from "lucide-svelte";
  import { routerStore } from "../../stores/router.svelte";
  import { groupStore } from "../../stores/groups.svelte";
  import { autofocusFirstButton } from "../../lib/actions";
  import type { ReadingGroup } from "../../types";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    group: ReadingGroup;
    groupId: string;
    isOwner: boolean;
    onDeleteError?: (message: string) => void;
  }

  let { group, groupId, isOwner, onDeleteError = () => {} }: Props = $props();

  let editing = $state(false);
  let editName = $state("");
  let editDescription = $state("");
  let saving = $state(false);
  let saveError: string | null = $state(null);
  let editNameInvalid = $derived(!!saveError);

  let confirmDelete = $state(false);
  let deleting = $state(false);

  $effect(() => {
    if (!groupId) return;
    editing = false;
    editName = "";
    editDescription = "";
    saving = false;
    saveError = null;
    confirmDelete = false;
    deleting = false;
  });

  function startEditing() {
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
    if (!editName.trim()) return;
    const capturedGroupId = groupId;
    saving = true;
    saveError = null;
    try {
      await groupStore.update(group.id, {
        name: editName.trim(),
        description: editDescription.trim() || null,
      });
      if (groupId !== capturedGroupId) return;
      editing = false;
    } catch (e) {
      if (groupId !== capturedGroupId) return;
      saveError = e instanceof Error ? e.message : "Failed to update group";
    } finally {
      if (groupId === capturedGroupId) saving = false;
    }
  }

  async function handleDelete() {
    const capturedGroupId = groupId;
    deleting = true;
    try {
      await groupStore.remove(group.id);
      routerStore.navigate("groups");
    } catch (e) {
      if (groupId !== capturedGroupId) return;
      onDeleteError(e instanceof Error ? e.message : "Failed to delete group");
      confirmDelete = false;
    } finally {
      if (groupId === capturedGroupId) deleting = false;
    }
  }
</script>

<div class="mb-8">
  {#if editing}
    <div
      class="p-5 bg-white dark:bg-ink-900 rounded-2xl border border-ink-200 dark:border-ink-700 shadow-sm mb-6"
    >
      <h2 class="text-lg font-semibold text-ink-900 dark:text-cream-100 mb-4">
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
            aria-describedby={editNameInvalid ? "edit-group-error" : undefined}
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
          <Button variant="secondary" onclick={cancelEditing} disabled={saving}>
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
      <p class="text-ink-600 dark:text-ink-300 mb-6">{group.description}</p>
    {/if}
  {/if}
</div>
