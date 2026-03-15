<script lang="ts">
  import { onMount } from "svelte";
  import { routerStore } from "../stores/router.svelte";
  import { getConfigStatus } from "../lib/api";
  import { Mail, Palette, Shield, Users } from "lucide-svelte";
  import AccountTab from "./settings/AccountTab.svelte";
  import OidcTab from "./settings/OidcTab.svelte";
  import UsersTab from "./settings/UsersTab.svelte";
  import PreferencesTab from "./settings/PreferencesTab.svelte";

  type SettingsTab =
    | "account"
    | "preferences"
    | "oidc"
    | "users";
  const validTabs: SettingsTab[] = [
    "account",
    "preferences",
    "oidc",
    "users",
  ];

  let activeTab: SettingsTab = $derived(
    validTabs.includes(routerStore.subPath as SettingsTab)
      ? (routerStore.subPath as SettingsTab)
      : "account",
  );

  let isAdmin = $state(false);
  let oidcConfigured = $state(false);

  onMount(async () => {
    try {
      const status = await getConfigStatus();
      oidcConfigured = status.oidc_configured;
      isAdmin = status.is_admin;
    } catch {
      // ignore - will show as not configured
    }
  });
</script>

<div>
  <div class="mb-6">
    <h1 class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100">
      Settings
    </h1>
    <p class="text-sm text-ink-400 dark:text-ink-400">
      Manage your account and preferences
    </p>
  </div>

  <div class="flex gap-6">
    <aside class="w-full sm:w-48 flex-shrink-0">
      <nav
        class="flex sm:flex-col gap-2 sm:gap-1 overflow-x-auto sm:overflow-x-visible"
      >
        <button
          onclick={() => routerStore.navigate("settings/account")}
          class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
          'account'
            ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
            : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
        >
          <Mail class="w-5 h-5" />
          Account
        </button>
        {#if isAdmin}
          <button
            onclick={() => routerStore.navigate("settings/oidc")}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
            'oidc'
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Shield class="w-5 h-5" />
            OIDC / SSO
          </button>
          <button
            onclick={() => routerStore.navigate("settings/users")}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
            'users'
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Users class="w-5 h-5" />
            Users
          </button>
        {/if}
        <button
          onclick={() => routerStore.navigate("settings/preferences")}
          class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
          'preferences'
            ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
            : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
        >
          <Palette class="w-5 h-5" />
          Preferences
        </button>
      </nav>
    </aside>

    <section class="flex-1">
      {#if activeTab === "account"}
        <AccountTab {oidcConfigured} />
      {/if}

      {#if activeTab === "oidc" && isAdmin}
        <OidcTab initialOidcConfigured={oidcConfigured} />
      {/if}

      {#if activeTab === "users" && isAdmin}
        <UsersTab />
      {/if}

      {#if activeTab === "preferences"}
        <PreferencesTab />
      {/if}
    </section>
  </div>
</div>
