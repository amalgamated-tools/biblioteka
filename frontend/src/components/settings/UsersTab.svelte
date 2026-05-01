<script lang="ts">
  import { authStore } from "../../stores/auth.svelte";
  import {
    listUsers,
    setUserAdmin,
    getRegistrationConfig,
    setRegistrationConfig,
  } from "../../lib/api";
  import type { AdminUser } from "../../types";
  import { Users, UserX } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";

  interface Props {
    cachedUsers: AdminUser[];
    onUsersLoaded: (users: AdminUser[]) => void;
  }

  let { cachedUsers, onUsersLoaded }: Props = $props();

  let userList: AdminUser[] = $state([]);
  let usersLoading = $state(false);
  let usersError: string | null = $state(null);
  let togglingId: string | null = $state(null);
  let hasFetchedUsers = false;

  // Registration config state
  let registrationDisabled = $state(false);
  let registrationLoading = $state(false);
  let registrationError: string | null = $state(null);
  let registrationSuccess: string | null = $state(null);

  $effect(() => {
    if (hasFetchedUsers) return;
    if (cachedUsers.length > 0) {
      userList = cachedUsers;
      hasFetchedUsers = true;
    } else if (!usersLoading) {
      void loadUsers();
    }
  });

  $effect(() => {
    void (async () => {
      try {
        const config = await getRegistrationConfig();
        registrationDisabled = config.registration_disabled;
      } catch {
        // ignore — admin-only; non-admins won't reach this tab
      }
    })();
  });

  async function loadUsers() {
    if (usersLoading || hasFetchedUsers) return;
    hasFetchedUsers = true;
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

  async function toggleRegistration() {
    registrationLoading = true;
    registrationError = null;
    registrationSuccess = null;
    try {
      const updated = await setRegistrationConfig({
        registration_disabled: !registrationDisabled,
      });
      registrationDisabled = updated.registration_disabled;
      registrationSuccess = registrationDisabled
        ? "Public registration disabled."
        : "Public registration enabled.";
    } catch (err) {
      registrationError =
        err instanceof Error
          ? err.message
          : "Failed to update registration setting";
    } finally {
      registrationLoading = false;
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
      <Users class="w-5 h-5 text-accent-600" aria-hidden="true" />
      User Management
    </h2>

    {#if usersLoading}
      <p role="status" class="text-ink-500 dark:text-ink-300">
        Loading users...
      </p>
    {:else if usersError}
      <AlertBanner variant="error">{usersError}</AlertBanner>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm" aria-label="Users">
          <thead>
            <tr
              class="text-left text-ink-500 dark:text-ink-300 border-b border-ink-100 dark:border-ink-800"
            >
              <th scope="col" class="pb-3 font-medium">Name</th>
              <th scope="col" class="pb-3 font-medium">Email</th>
              <th scope="col" class="pb-3 font-medium">Type</th>
              <th scope="col" class="pb-3 font-medium">Role</th>
              <th scope="col" class="pb-3 font-medium">Joined</th>
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
                      aria-label={u.is_admin
                        ? `Remove admin role from ${u.name || u.email}`
                        : `Grant admin role to ${u.name || u.email}`}
                      class="px-3 py-1 rounded-full text-xs font-medium transition-colors disabled:opacity-50 {u.is_admin
                        ? 'bg-success-50 text-success-700 hover:bg-danger-50 hover:text-danger-700 dark:bg-green-900/20 dark:text-green-400 dark:hover:bg-danger-700/10 dark:hover:text-red-400'
                        : 'bg-ink-50 text-ink-500 hover:bg-success-50 hover:text-success-700 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-green-900/20 dark:hover:text-green-400'}"
                    >
                      {u.is_admin ? "Admin" : "User"}
                    </button>
                  {/if}
                </td>
                <td class="py-3 text-ink-500 dark:text-ink-300"
                  >{new Date(u.created_at).toLocaleDateString()}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <div
    class="border-t border-ink-100 dark:border-ink-800 pt-6"
    aria-labelledby="registration-heading"
  >
    <h3
      id="registration-heading"
      class="text-base font-semibold text-ink-900 dark:text-cream-100 mb-1 flex items-center gap-2"
    >
      <UserX class="w-4 h-4 text-accent-600" aria-hidden="true" />
      Public Registration
    </h3>
    <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
      When disabled, new users cannot sign up through the public registration
      form. Admins can still create accounts via the admin panel.
    </p>

    {#if registrationError}
      <AlertBanner variant="error">{registrationError}</AlertBanner>
    {/if}
    {#if registrationSuccess}
      <AlertBanner variant="success">{registrationSuccess}</AlertBanner>
    {/if}

    <div class="flex items-center gap-4">
      <Button
        variant={registrationDisabled ? "primary" : "secondary"}
        disabled={registrationLoading}
        onclick={toggleRegistration}
      >
        {registrationLoading
          ? "Saving..."
          : registrationDisabled
            ? "Enable Registration"
            : "Disable Registration"}
      </Button>
      <span
        class="text-sm {registrationDisabled
          ? 'text-danger-600 dark:text-red-400'
          : 'text-success-700 dark:text-green-400'}"
      >
        {registrationDisabled
          ? "Registration is disabled"
          : "Registration is enabled"}
      </span>
    </div>
  </div>
</div>
