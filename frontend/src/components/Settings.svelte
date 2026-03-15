<script lang="ts">
  import { onMount } from "svelte";
  import { routerStore } from "../stores/router.svelte";
  import {
    getConfigStatus,
    getOidcConfig,
    getSmtpConfig,
    setSmtpConfig,
    testSmtpConfig,
    type AdminUser,
  } from "../lib/api";
  import { Mail, Palette, Shield, Users, Send } from "lucide-svelte";
  import AccountTab from "./settings/AccountTab.svelte";
  import OidcTab from "./settings/OidcTab.svelte";
  import UsersTab from "./settings/UsersTab.svelte";
  import PreferencesTab from "./settings/PreferencesTab.svelte";

  type SettingsTab = "account" | "preferences" | "oidc" | "smtp" | "users";
  const validTabs: SettingsTab[] = [
    "account",
    "preferences",
    "oidc",
    "smtp",
    "users",
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

  // SMTP state
  let smtpConfigured = $state(false);
  let smtpEnvOverride = $state(false);
  let smtpPasswordSet = $state(false);
  let smtpHost = $state("");
  let smtpPort = $state("587");
  let smtpUsername = $state("");
  let smtpPassword = $state("");
  let smtpFrom = $state("");
  let smtpTls = $state("starttls");
  let smtpError: string | null = $state(null);
  let smtpSuccess = $state(false);
  let smtpLoading = $state(false);
  let smtpTestLoading = $state(false);
  let smtpTestMessage: string | null = $state(null);
  let smtpTestError: string | null = $state(null);

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
      if (status.is_admin) {
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

  async function handleSmtpSave(e: SubmitEvent) {
    e.preventDefault();
    smtpError = null;
    smtpSuccess = false;

    if (!smtpHost.trim()) {
      smtpError = "SMTP Host is required";
      return;
    }
    if (!smtpFrom.trim()) {
      smtpError = "From Address is required";
      return;
    }
    if (!smtpPassword.trim() && !smtpPasswordSet && !smtpEnvOverride && smtpUsername.trim()) {
      smtpError = "Password is required when username is set";
      return;
    }

    smtpLoading = true;

    try {
      await setSmtpConfig({
        host: smtpHost.trim(),
        port: smtpPort.trim() || "587",
        username: smtpUsername.trim(),
        password: smtpPassword.trim(),
        from: smtpFrom.trim(),
        tls: smtpTls,
      });
      smtpSuccess = true;
      smtpConfigured = true;
      if (smtpPassword.trim()) {
        smtpPasswordSet = true;
      }
      smtpPassword = "";
      setTimeout(() => (smtpSuccess = false), 3000);
    } catch (err) {
      smtpError =
        err instanceof Error
          ? err.message
          : "Failed to save SMTP configuration";
    } finally {
      smtpLoading = false;
    }
  }

  async function handleSmtpTest() {
    smtpTestMessage = null;
    smtpTestError = null;
    smtpTestLoading = true;

    try {
      const result = await testSmtpConfig();
      smtpTestMessage = result.message;
      setTimeout(() => (smtpTestMessage = null), 5000);
    } catch (err) {
      smtpTestError =
        err instanceof Error ? err.message : "Failed to send test email";
      setTimeout(() => (smtpTestError = null), 5000);
    } finally {
      smtpTestLoading = false;
    }
  }
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
        <div
          class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
        >
          <div>
            <h2
              class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
            >
              <Send class="w-5 h-5 text-accent-600" />
              Email / SMTP Configuration
            </h2>

            <div class="mb-4">
              <div class="flex items-center gap-2 text-sm">
                <span class="text-ink-500">Status:</span>
                {#if smtpConfigured}
                  <span
                    class="inline-flex items-center gap-1.5 text-success-700 dark:text-green-400 bg-success-50 dark:bg-green-900/20 px-2.5 py-1 rounded-full font-medium"
                  >
                    <span class="w-2 h-2 rounded-full bg-success-600"></span>
                    Configured
                  </span>
                {:else}
                  <span
                    class="inline-flex items-center gap-1.5 text-ink-600 dark:text-ink-300 bg-ink-50 dark:bg-ink-800 px-2.5 py-1 rounded-full font-medium"
                  >
                    <span class="w-2 h-2 rounded-full bg-ink-300"></span>
                    Not configured
                  </span>
                {/if}
              </div>
            </div>

            <p class="text-sm text-ink-500 dark:text-ink-400 mb-4">
              Configure SMTP settings to enable email notifications from
              Biblioteka.
            </p>

            {#if smtpEnvOverride}
              <div
                class="bg-accent-50 dark:bg-accent-800/20 border border-accent-200 dark:border-accent-700/30 text-accent-700 dark:text-accent-400 px-4 py-3 rounded-xl text-sm mb-4"
              >
                SMTP is currently configured via environment variables
                (SMTP_HOST, etc.). The values shown below reflect the active
                configuration. To use database-managed settings instead, remove
                the SMTP environment variables from the server.
              </div>
            {/if}

            <form onsubmit={handleSmtpSave} class="space-y-4">
              <div>
                <label
                  for="smtp-host"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  SMTP Host
                </label>
                <input
                  id="smtp-host"
                  type="text"
                  bind:value={smtpHost}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="smtp.example.com"
                  disabled={smtpLoading}
                />
              </div>

              <div>
                <label
                  for="smtp-port"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Port
                </label>
                <input
                  id="smtp-port"
                  type="text"
                  bind:value={smtpPort}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="587"
                  disabled={smtpLoading}
                />
              </div>

              <div>
                <label
                  for="smtp-username"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Username
                </label>
                <input
                  id="smtp-username"
                  type="text"
                  bind:value={smtpUsername}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="user@example.com"
                  disabled={smtpLoading}
                />
              </div>

              <div>
                <label
                  for="smtp-password"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Password
                </label>
                <input
                  id="smtp-password"
                  type="password"
                  bind:value={smtpPassword}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder={smtpPasswordSet || smtpEnvOverride
                    ? "Enter new password to update"
                    : "Enter your SMTP password"}
                  disabled={smtpLoading}
                />
                {#if smtpPasswordSet || smtpEnvOverride}
                  <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
                    {smtpEnvOverride
                      ? "Password is configured via environment variable"
                      : "Leave blank to keep the existing password"}
                  </p>
                {/if}
              </div>

              <div>
                <label
                  for="smtp-from"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  From Address
                </label>
                <input
                  id="smtp-from"
                  type="email"
                  bind:value={smtpFrom}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="noreply@example.com"
                  disabled={smtpLoading}
                />
                <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
                  The email address that outgoing messages will be sent from
                </p>
              </div>

              <div>
                <label
                  for="smtp-tls"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  TLS Mode
                </label>
                <select
                  id="smtp-tls"
                  bind:value={smtpTls}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  disabled={smtpLoading}
                >
                  <option value="starttls">STARTTLS (port 587)</option>
                  <option value="tls">TLS (port 465)</option>
                  <option value="none">None (port 25)</option>
                </select>
              </div>

              {#if smtpError}
                <div
                  class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
                >
                  {smtpError}
                </div>
              {/if}

              {#if smtpSuccess}
                <div
                  class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
                >
                  SMTP configuration saved successfully
                </div>
              {/if}

              <button
                type="submit"
                disabled={smtpLoading}
                class="w-full px-4 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
              >
                {smtpLoading
                  ? "Saving..."
                  : smtpConfigured
                    ? "Update Configuration"
                    : "Save Configuration"}
              </button>
            </form>

            {#if smtpConfigured}
              <hr class="border-ink-100 dark:border-ink-800" />

              <div>
                <h3
                  class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-3"
                >
                  Test Configuration
                </h3>
                <p class="text-sm text-ink-500 dark:text-ink-400 mb-4">
                  Send a test email to your account email address to verify the
                  SMTP settings are working correctly.
                </p>

                {#if smtpTestMessage}
                  <div
                    class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm animate-scale-in mb-4"
                  >
                    {smtpTestMessage}
                  </div>
                {/if}

                {#if smtpTestError}
                  <div
                    class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in mb-4"
                  >
                    {smtpTestError}
                  </div>
                {/if}

                <button
                  onclick={handleSmtpTest}
                  disabled={smtpTestLoading}
                  class="inline-flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
                >
                  <Mail class="w-4 h-4" />
                  {smtpTestLoading ? "Sending..." : "Send Test Email"}
                </button>
              </div>
            {/if}
          </div>
        </div>
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
