<script lang="ts">
  import { onMount } from "svelte";
  import { authStore } from "../../stores/auth.svelte";
  import { listUsers, setUserAdmin, type AdminUser } from "../../lib/api";
  import { Users } from "lucide-svelte";

  interface Props {
    cachedUsers: AdminUser[];
    onUsersLoaded: (users: AdminUser[]) => void;
  }

  let { cachedUsers, onUsersLoaded }: Props = $props();

  // One-time initialisation – cachedUsers seeds the list; subsequent updates
  // come from loadUsers() / toggleAdmin(), not from prop changes.
  // svelte-ignore state_referenced_locally
  let userList: AdminUser[] = $state.raw(cachedUsers);
  let usersLoading = $state(false);
  let usersError: string | null = $state(null);
  let togglingId: string | null = $state(null);

  $effect(() => {
    if (userList.length === 0 && cachedUsers.length > 0) {
      userList = cachedUsers;
    }
  });

  onMount(() => {
    if (userList.length === 0) {
      loadUsers();
    }
  });

  async function loadUsers() {
    usersLoading = true;
    usersError = null;
    try {
      userList = await listUsers();
      onUsersLoaded(userList);
    } catch (err) {
      usersError = err instanceof Error ? err.message : "Failed to load users";
    } finally {
      usersLoading = false;
    }
  }

  async function toggleAdmin(u: AdminUser) {
    if (togglingId === u.id) return;
    togglingId = u.id;
    try {
      await setUserAdmin(u.id, !u.is_admin);
      userList = userList.map((item) =>
        item.id === u.id ? { ...item, is_admin: !item.is_admin } : item,
      );
      onUsersLoaded(userList);
    } catch (err) {
      usersError = err instanceof Error ? err.message : "Failed to update user";
    } finally {
      togglingId = null;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
    >
      <Users class="w-5 h-5 text-accent-600" />
      User Management
    </h2>

    {#if usersLoading}
      <p class="text-ink-400 dark:text-ink-300">Loading users...</p>
    {:else if usersError}
      <div
        class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm"
      >
        {usersError}
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr
              class="text-left text-ink-400 dark:text-ink-300 border-b border-ink-100 dark:border-ink-800"
            >
              <th class="pb-3 font-medium">Name</th>
              <th class="pb-3 font-medium">Email</th>
              <th class="pb-3 font-medium">Type</th>
              <th class="pb-3 font-medium">Role</th>
              <th class="pb-3 font-medium">Joined</th>
            </tr>
          </thead>
          <tbody>
            {#each userList as u (u.id)}
              <tr
                class="border-b border-ink-50 dark:border-ink-800 hover:bg-ink-50/50 dark:hover:bg-ink-800/50 transition-colors"
              >
                <td class="py-3 text-ink-900 dark:text-cream-100 font-medium"
                  >{u.name}</td
                >
                <td class="py-3 text-ink-500 dark:text-ink-300">{u.email}</td>
                <td class="py-3">
                  <span
                    class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium {u.oidc_linked
                      ? 'bg-accent-50 text-accent-700 dark:bg-accent-800/20 dark:text-accent-400'
                      : 'bg-ink-50 text-ink-500 dark:bg-ink-800 dark:text-ink-300'}"
                  >
                    {u.oidc_linked ? "OIDC/SSO" : "Local"}
                  </span>
                </td>
                <td class="py-3">
                  {#if u.id === authStore.user?.id}
                    <span
                      class="inline-flex items-center gap-1.5 text-success-700 dark:text-green-400 bg-success-50 dark:bg-green-900/20 px-2.5 py-1 rounded-full text-xs font-medium"
                    >
                      Admin (you)
                    </span>
                  {:else}
                    <button
                      onclick={() => toggleAdmin(u)}
                      disabled={togglingId === u.id}
                      class="px-3 py-1 rounded-full text-xs font-medium transition-colors disabled:opacity-50 {u.is_admin
                        ? 'bg-success-50 text-success-700 hover:bg-danger-50 hover:text-danger-700 dark:bg-green-900/20 dark:text-green-400 dark:hover:bg-danger-700/10 dark:hover:text-red-400'
                        : 'bg-ink-50 text-ink-500 hover:bg-success-50 hover:text-success-700 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-green-900/20 dark:hover:text-green-400'}"
                    >
                      {u.is_admin ? "Admin" : "User"}
                    </button>
                  {/if}
                </td>
                <td class="py-3 text-ink-400 dark:text-ink-500"
                  >{new Date(u.created_at).toLocaleDateString()}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
