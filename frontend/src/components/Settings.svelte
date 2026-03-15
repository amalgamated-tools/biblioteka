<script lang="ts">
  import { onMount } from "svelte";
  import { routerStore } from "../stores/router.svelte";
  import {
    getConfigStatus,
    getOidcConfig,
    getSmtpConfig,
    type AdminUser,
  } from "../lib/api";
  import { Mail, Palette, Shield, Users, Send, KeyRound } from "lucide-svelte";
  import AccountTab from "./settings/AccountTab.svelte";
  import OidcTab from "./settings/OidcTab.svelte";
  import UsersTab from "./settings/UsersTab.svelte";
  import PreferencesTab from "./settings/PreferencesTab.svelte";
  import SmtpTab from "./settings/SmtpTab.svelte";
  import APIKeysTab from "./settings/APIKeysTab.svelte";

  type SettingsTab = "account" | "preferences" | "oidc" | "smtp" | "users" | "api-keys";
  const validTabs: SettingsTab[] = [
    "account",
    "preferences",
    "oidc",
    "smtp",
    "users",
    "api-keys",
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

  // SMTP initial state (populated in onMount, passed to SmtpTab as initial props)
  let smtpConfigured = $state(false);
  let smtpEnvOverride = $state(false);
  let smtpPasswordSet = $state(false);
  let smtpHost = $state("");
  let smtpPort = $state("587");
  let smtpUsername = $state("");
  let smtpFrom = $state("");
  let smtpTls = $state("starttls");

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
        try {
          const smtp = await getSmtpConfig();
          smtpHost = smtp.host;
          smtpPort = smtp.port || "587";
          smtpUsername = smtp.username;
          smtpFrom = smtp.from;
          smtpTls = smtp.tls || "starttls";
          smtpEnvOverride = smtp.env_override;
          smtpPasswordSet = smtp.password_set;
        } catch {
          // ignore - user can re-enter
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
        <button
          onclick={() => routerStore.navigate("settings/api-keys")}
          class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
          'api-keys'
            ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
            : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
        >
          <KeyRound class="w-5 h-5" />
          API Keys
        </button>
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
            onclick={() => routerStore.navigate("settings/smtp")}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
            'smtp'
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Send class="w-5 h-5" />
            Email / SMTP
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
      </nav>
    </aside>

    <section class="flex-1">
      {#if activeTab === "account"}
        <AccountTab {oidcConfigured} />
      {/if}

      {#if activeTab === "api-keys"}
        <APIKeysTab />
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
        <SmtpTab
          initialConfigured={smtpConfigured}
          initialEnvOverride={smtpEnvOverride}
          initialPasswordSet={smtpPasswordSet}
          initialHost={smtpHost}
          initialPort={smtpPort}
          initialUsername={smtpUsername}
          initialFrom={smtpFrom}
          initialTls={smtpTls}
        />
      {/if}

      {#if activeTab === "users" && isAdmin}
        <UsersTab
          {cachedUsers}
          onUsersLoaded={(users) => (cachedUsers = users)}
        />
      {/if}

      {#if activeTab === "preferences"}
        <PreferencesTab />
      {/if}
    </section>
  </div>
</div>
