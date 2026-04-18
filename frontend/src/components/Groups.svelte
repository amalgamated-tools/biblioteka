<script lang="ts">
  import { Users, Plus, X, Check } from "lucide-svelte";
  import { routerStore } from "../stores/router.svelte";
  import { groupStore } from "../stores/groups.svelte";
  import AlertBanner from "./ui/AlertBanner.svelte";
  import Button from "./ui/Button.svelte";
  import TextInput from "./ui/TextInput.svelte";
  import GroupDetail from "./groups/GroupDetail.svelte";

  let error = $derived(groupStore.loadError);
  let showForm = $state(false);
  let newName = $state("");
  let newDescription = $state("");
  let creating = $state(false);
  let createError: string | null = $state(null);
  let createNameInvalid = $derived(!!createError);

  $effect(() => {
    if (!groupStore.loaded && !groupStore.loading) {
      void groupStore.load();
    }
  });

  let subPath = $derived(routerStore.subPath);
  let viewingGroup = $derived(subPath !== "" && subPath !== "new");

  $effect(() => {
    if (subPath === "new") {
      showForm = true;
    } else if (!viewingGroup) {
      showForm = false;
      newName = "";
      newDescription = "";
      createError = null;
    }
  });

  async function handleCreate() {
    if (!newName.trim()) return;
    creating = true;
    createError = null;
    try {
      const created = await groupStore.create({
        name: newName.trim(),
        description: newDescription.trim() || null,
      });
      newName = "";
      newDescription = "";
      showForm = false;
      routerStore.navigate(`groups/${created.id}`);
    } catch (e) {
      createError = e instanceof Error ? e.message : "Failed to create group";
    } finally {
      creating = false;
    }
  }

  function cancelCreate() {
    showForm = false;
    newName = "";
    newDescription = "";
    createError = null;
    if (subPath === "new") {
      routerStore.navigate("groups");
    }
  }
</script>

{#if viewingGroup}
  <GroupDetail groupId={subPath} />
{:else}
  <div>
    <div class="flex items-center justify-between mb-8">
      <div class="flex items-center gap-3">
        <div
          class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center"
        >
          <Users
            class="w-5 h-5 text-accent-600 dark:text-accent-400"
            aria-hidden="true"
          />
        </div>
        <h1
          class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100"
        >
          Reading Groups
        </h1>
      </div>
      {#if !showForm}
        <Button onclick={() => (showForm = true)}>
          <Plus class="w-4 h-4 mr-1" aria-hidden="true" />
          New Group
        </Button>
      {/if}
    </div>

    {#if error}
      <AlertBanner variant="error" class="mb-4">{error}</AlertBanner>
    {/if}

    {#if showForm}
      <div
        class="mb-6 p-5 bg-white dark:bg-ink-900 rounded-2xl border border-ink-200 dark:border-ink-700 shadow-sm"
      >
        <h2 class="text-lg font-semibold text-ink-900 dark:text-cream-100 mb-4">
          Create Reading Group
        </h2>
        {#if createError}
          <AlertBanner id="create-group-error" variant="error" class="mb-3"
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
              for="new-group-name"
              class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
            >
              Name <span class="text-danger-500" aria-hidden="true">*</span>
            </label>
            <TextInput
              id="new-group-name"
              bind:value={newName}
              placeholder="e.g. Book Club, Sci-Fi Friends…"
              disabled={creating}
              aria-required={true}
              aria-invalid={createNameInvalid}
              aria-describedby={createNameInvalid
                ? "create-group-error"
                : undefined}
            />
          </div>
          <div>
            <label
              for="new-group-desc"
              class="block text-sm font-medium text-ink-700 dark:text-ink-300 mb-1"
            >
              Description
            </label>
            <TextInput
              id="new-group-desc"
              bind:value={newDescription}
              placeholder="Optional description"
              disabled={creating}
            />
          </div>
          <div class="flex gap-2 pt-1">
            <Button
              onclick={handleCreate}
              disabled={creating || !newName.trim()}
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

    {#if groupStore.groups.length === 0 && groupStore.loaded && !groupStore.loadError}
      <div
        class="flex flex-col items-center justify-center py-20 text-center"
        aria-live="polite"
      >
        <div
          class="w-16 h-16 bg-ink-100 dark:bg-ink-800 rounded-2xl flex items-center justify-center mb-4"
        >
          <Users
            class="w-8 h-8 text-ink-400 dark:text-ink-500"
            aria-hidden="true"
          />
        </div>
        <h2 class="text-xl font-semibold text-ink-700 dark:text-ink-300 mb-2">
          No reading groups yet
        </h2>
        <p class="text-ink-500 dark:text-ink-400 mb-6">
          Create a group to read and discuss books with others.
        </p>
        <Button onclick={() => (showForm = true)}>
          <Plus class="w-4 h-4 mr-1" aria-hidden="true" />
          Create Your First Group
        </Button>
      </div>
    {:else}
      <ul class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" role="list">
        {#each groupStore.groups as group (group.id)}
          <li>
            <a
              href={`#groups/${group.id}`}
              class="group block p-5 bg-white dark:bg-ink-900 rounded-2xl border border-ink-200 dark:border-ink-700 shadow-sm hover:shadow-md hover:border-accent-300 dark:hover:border-accent-700 transition-all"
              aria-label={`View ${group.name}`}
            >
              <div class="flex items-start justify-between gap-2">
                <div class="flex items-center gap-3 min-w-0">
                  <div
                    class="w-9 h-9 bg-accent-100 dark:bg-accent-800/30 rounded-lg flex items-center justify-center flex-shrink-0"
                  >
                    <Users
                      class="w-4 h-4 text-accent-600 dark:text-accent-400"
                      aria-hidden="true"
                    />
                  </div>
                  <div class="min-w-0">
                    <p
                      class="font-semibold text-ink-900 dark:text-cream-100 truncate group-hover:text-accent-600 dark:group-hover:text-accent-400 transition-colors"
                    >
                      {group.name}
                    </p>
                    <p class="text-xs text-ink-500 dark:text-ink-400 mt-0.5">
                      {group.member_count}
                      {group.member_count === 1 ? "member" : "members"}
                    </p>
                  </div>
                </div>
              </div>
              {#if group.description}
                <p
                  class="mt-3 text-sm text-ink-600 dark:text-ink-300 line-clamp-2"
                >
                  {group.description}
                </p>
              {/if}
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{/if}
