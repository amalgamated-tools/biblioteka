<script lang="ts">
  import { onMount } from "svelte";
  import { routerStore } from "../stores/router.svelte";
  import { getConfigStatus, getOidcConfig, type AdminUser } from "../lib/api";
  import {
    Mail,
    Palette,
    Shield,
    Users,
    Send,
    KeyRound,
    BookOpen,
  } from "lucide-svelte";

  const userTabs: { key: string; label: string; icon: typeof Mail }[] = [
    { key: "account", label: "Account", icon: Mail },
    { key: "api-keys", label: "API Keys", icon: KeyRound },
    { key: "kobo", label: "Kobo Sync", icon: BookOpen },
    { key: "preferences", label: "Preferences", icon: Palette },
  ];
  import AccountTab from "./settings/AccountTab.svelte";
  import OidcTab from "./settings/OidcTab.svelte";
  import SmtpTab from "./settings/SmtpTab.svelte";
  import UsersTab from "./settings/UsersTab.svelte";
  import PreferencesTab from "./settings/PreferencesTab.svelte";
  import APIKeysTab from "./settings/APIKeysTab.svelte";
  import KoboTab from "./settings/KoboTab.svelte";

  type SettingsTab =
    | "account"
    | "preferences"
    | "oidc"
    | "smtp"
    | "users"
    | "api-keys"
    | "kobo";

  const userTabLabels = userTabs.reduce(
    (acc, tab) => {
      acc[tab.key as SettingsTab] = tab.label;
      return acc;
    },
    {} as Partial<Record<SettingsTab, string>>,
  );

  const tabLabels: Record<SettingsTab, string> = {
    ...(userTabLabels as Record<SettingsTab, string>),
    oidc: "OIDC / SSO",
    smtp: "Email / SMTP",
    users: "Users",
  };
  const validTabs: SettingsTab[] = [
    "account",
    "preferences",
    "oidc",
    "smtp",
    "users",
    "api-keys",
    "kobo",
  ];

  let activeTab: SettingsTab = $derived(
    validTabs.includes(routerStore.subPath as SettingsTab)
      ? (routerStore.subPath as SettingsTab)
      : "account",
  );

  let isAdmin = $state(false);
  let oidcConfigured = $state(false);
  let oidcIssuerUrl = $state("");
  let oidcClientId = $state("");
  let oidcRedirectUri = $state("");
  let cachedUsers: AdminUser[] = $state.raw([]);
  let smtpConfigured = $state(false);

  onMount(async () => {
    try {
      const status = await getConfigStatus();
      oidcConfigured = status.oidc_configured;
      smtpConfigured = status.smtp_configured;
      isAdmin = status.is_admin;

      if (isAdmin) {
        const oidcConfig = status.oidc_configured
          ? await getOidcConfig()
          : null;
        if (oidcConfig) {
          oidcIssuerUrl = oidcConfig.issuer_url;
          oidcClientId = oidcConfig.client_id;
          oidcRedirectUri = oidcConfig.redirect_uri;
        }
      }
    } catch {
      // ignore - will show as not configured
    }
  });
</script>

<div>
  <div class="mb-6">
    <h1
      class="text-2xl font-display font-bold text-ink-900 dark:text-cream-100"
    >
      Settings
    </h1>
    <p class="text-sm text-ink-400 dark:text-ink-400">
      Manage your account and preferences
    </p>
  </div>

  <div class="flex gap-6">
    <div class="w-full sm:w-48 flex-shrink-0">
      <nav
        aria-label="Settings sections"
        class="flex sm:flex-col gap-2 sm:gap-1 overflow-x-auto sm:overflow-x-visible"
      >
        {#each userTabs as tab (tab.key)}
          {@const isActive = activeTab === tab.key}
          <a
            href="#settings/{tab.key}"
            aria-current={isActive ? "page" : undefined}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {isActive
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <tab.icon class="w-5 h-5" />
            {tab.label}
          </a>
        {/each}
        {#if isAdmin}
          <div class="hidden sm:flex items-center gap-2 px-4 pt-3 pb-1">
            <hr class="flex-1 border-ink-200 dark:border-ink-700" />
            <span
              class="text-xs font-medium uppercase text-ink-400 dark:text-ink-500"
              >Admin</span
            >
            <hr class="flex-1 border-ink-200 dark:border-ink-700" />
          </div>
          <div
            class="sm:hidden w-px bg-ink-200 dark:bg-ink-700 self-stretch my-1"
          ></div>
          {@const isOidcActive = activeTab === "oidc"}
          <a
            href="#settings/oidc"
            aria-current={isOidcActive ? "page" : undefined}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {isOidcActive
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Shield class="w-5 h-5" />
            OIDC / SSO
          </a>
          {@const isSmtpActive = activeTab === "smtp"}
          <a
            href="#settings/smtp"
            aria-current={isSmtpActive ? "page" : undefined}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {isSmtpActive
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Send class="w-5 h-5" />
            Email / SMTP
          </a>
          {@const isUsersActive = activeTab === "users"}
          <a
            href="#settings/users"
            aria-current={isUsersActive ? "page" : undefined}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {isUsersActive
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Users class="w-5 h-5" />
            Users
          </a>
        {/if}
      </nav>
    </div>

    <section class="flex-1" aria-label="{tabLabels[activeTab]} settings">
      {#if activeTab === "account"}
        <AccountTab {oidcConfigured} />
      {/if}

      {#if activeTab === "oidc" && isAdmin}
        <OidcTab
          initialOidcConfigured={oidcConfigured}
          initialIssuerUrl={oidcIssuerUrl}
          initialClientId={oidcClientId}
          initialRedirectUri={oidcRedirectUri}
          onOidcSaved={(cfg) => {
            oidcConfigured = cfg.configured;
            oidcIssuerUrl = cfg.issuerUrl;
            oidcClientId = cfg.clientId;
            oidcRedirectUri = cfg.redirectUri;
          }}
        />
      {/if}

      {#if activeTab === "smtp" && isAdmin}
        <SmtpTab initialSmtpConfigured={smtpConfigured} />
      {/if}

      {#if activeTab === "users" && isAdmin}
        <UsersTab
          {cachedUsers}
          onUsersLoaded={(users) => (cachedUsers = users)}
        />
      {/if}

      {#if activeTab === "api-keys"}
        <APIKeysTab />
      {/if}

      {#if activeTab === "kobo"}
        <KoboTab />
      {/if}

      {#if activeTab === "preferences"}
        <PreferencesTab />
      {/if}
    </section>
  </div>
</div>
