<script lang="ts">
  import { onMount } from "svelte";
  import { user, oidcLinkError } from "../stores/auth";
  import { themePreference, setTheme } from "../stores/theme";
  import {
    changePassword,
    getConfigStatus,
    getOidcConfig,
    setOidcConfig,
    createOidcLinkNonce,
    listUsers,
    setUserAdmin,
    type AdminUser,
  } from "../lib/api";
  import {
    Lock,
    Mail,
    Palette,
    Shield,
    Link,
    Users,
  } from "lucide-svelte";
  import { subPath, navigate } from "../stores/router";

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
    validTabs.includes($subPath as SettingsTab)
      ? ($subPath as SettingsTab)
      : "account",
  );
  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let passwordError: string | null = $state(null);
  let passwordSuccess = $state(false);
  let passwordLoading = $state(false);
  let compactView = $state(false);

  // OIDC state
  let isAdmin = $state(false);
  let oidcConfigured = $state(false);
  let oidcIssuerUrl = $state("");
  let oidcClientId = $state("");
  let oidcClientSecret = $state("");
  let oidcRedirectUri = $state("");
  let oidcError: string | null = $state(null);
  let oidcSuccess = $state(false);
  let oidcLoading = $state(false);
  let oidcStatusLoading = $state(true);

  // SSO link state
  let linkSsoLoading = $state(false);

  // User management state
  let userList: AdminUser[] = $state([]);
  let usersLoading = $state(false);
  let usersError: string | null = $state(null);

  $effect(() => {
    if (
      activeTab === "users" &&
      isAdmin &&
      userList.length === 0 &&
      !usersLoading
    ) {
      loadUsers();
    }
  });

  onMount(async () => {
    try {
      const status = await getConfigStatus();
      oidcConfigured = status.oidc_configured;
      isAdmin = status.is_admin;
      if (status.is_admin && status.oidc_configured) {
        try {
          const config = await getOidcConfig();
          oidcIssuerUrl = config.issuer_url;
          oidcClientId = config.client_id;
          oidcRedirectUri = config.redirect_uri;
        } catch {
          // ignore - user can re-enter
        }
      }
    } catch {
      // ignore - will show as not configured
    } finally {
      oidcStatusLoading = false;
    }
  });

  async function handleLinkSso() {
    linkSsoLoading = true;
    oidcLinkError.set(null);
    try {
      const nonce = await createOidcLinkNonce();
      window.location.href = `/api/auth/oidc/link?nonce=${encodeURIComponent(nonce)}`;
    } catch (err) {
      oidcLinkError.set(
        err instanceof Error ? err.message : "Failed to start SSO linking",
      );
      linkSsoLoading = false;
    }
  }

  async function handlePasswordChange(e: SubmitEvent) {
    e.preventDefault();
    passwordError = null;
    passwordSuccess = false;

    if (!currentPassword) {
      passwordError = "Current password is required";
      return;
    }

    if (!newPassword) {
      passwordError = "New password is required";
      return;
    }

    if (newPassword.length < 6) {
      passwordError = "New password must be at least 6 characters";
      return;
    }

    if (newPassword !== confirmPassword) {
      passwordError = "Passwords do not match";
      return;
    }

    passwordLoading = true;

    try {
      await changePassword(currentPassword, newPassword);
      passwordSuccess = true;
      currentPassword = "";
      newPassword = "";
      confirmPassword = "";
      setTimeout(() => (passwordSuccess = false), 3000);
    } catch (err) {
      passwordError =
        err instanceof Error ? err.message : "Failed to update password";
    } finally {
      passwordLoading = false;
    }
  }

  const themes = ["light", "dark", "auto"] as const;

  async function loadUsers() {
    usersLoading = true;
    usersError = null;
    try {
      userList = await listUsers();
    } catch (err) {
      usersError = err instanceof Error ? err.message : "Failed to load users";
    } finally {
      usersLoading = false;
    }
  }

  async function toggleAdmin(u: AdminUser) {
    try {
      await setUserAdmin(u.id, !u.is_admin);
      u.is_admin = !u.is_admin;
      userList = [...userList];
    } catch (err) {
      usersError = err instanceof Error ? err.message : "Failed to update user";
    }
  }

  async function handleOidcSave(e: SubmitEvent) {
    e.preventDefault();
    oidcError = null;
    oidcSuccess = false;

    if (!oidcIssuerUrl.trim()) {
      oidcError = "Issuer URL is required";
      return;
    }
    if (!oidcClientId.trim()) {
      oidcError = "Client ID is required";
      return;
    }
    if (!oidcClientSecret.trim() && !oidcConfigured) {
      oidcError = "Client Secret is required";
      return;
    }
    if (!oidcRedirectUri.trim()) {
      oidcError = "Redirect URI is required";
      return;
    }

    oidcLoading = true;

    try {
      await setOidcConfig({
        issuer_url: oidcIssuerUrl.trim(),
        client_id: oidcClientId.trim(),
        client_secret: oidcClientSecret.trim(),
        redirect_uri: oidcRedirectUri.trim(),
      });
      oidcSuccess = true;
      oidcConfigured = true;
      oidcClientSecret = "";
      setTimeout(() => (oidcSuccess = false), 3000);
    } catch (err) {
      oidcError =
        err instanceof Error
          ? err.message
          : "Failed to save OIDC configuration";
    } finally {
      oidcLoading = false;
    }
  }
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
          onclick={() => navigate("settings/account")}
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
            onclick={() => navigate("settings/oidc")}
            class="flex items-center gap-3 px-4 py-3 rounded-xl font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
            'oidc'
              ? 'bg-accent-50 text-accent-700 border-l-4 border-accent-600 dark:bg-accent-800/20 dark:text-accent-400'
              : 'text-ink-500 hover:bg-ink-50 dark:text-ink-400 dark:hover:bg-ink-800'}"
          >
            <Shield class="w-5 h-5" />
            OIDC / SSO
          </button>
          <button
            onclick={() => navigate("settings/users")}
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
          onclick={() => navigate("settings/preferences")}
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
        <div
          class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
        >
          <div>
            <h2
              class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
            >
              <Mail class="w-5 h-5 text-accent-600" />
              Account Information
            </h2>
            <div class="space-y-4">
              <div>
                <label
                  for="email-display"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Email Address
                </label>
                <input
                  id="email-display"
                  type="email"
                  value={$user?.email || ""}
                  disabled
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl bg-ink-50 dark:bg-ink-800 text-ink-500 dark:text-ink-400 cursor-not-allowed"
                />
                <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
                  Contact support to change your email address
                </p>
              </div>
            </div>
          </div>

          <hr class="border-ink-100 dark:border-ink-800" />

          {#if oidcConfigured}
            <div>
              <h2
                class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
              >
                <Link class="w-5 h-5 text-accent-600" />
                Single Sign-On
              </h2>
              {#if $user?.oidc_linked}
                <div class="flex items-center gap-2">
                  <span
                    class="inline-flex items-center gap-1.5 text-success-700 dark:text-green-400 bg-success-50 dark:bg-green-900/20 px-2.5 py-1 rounded-full text-sm font-medium"
                  >
                    <span class="w-2 h-2 rounded-full bg-success-600"></span>
                    SSO Connected
                  </span>
                </div>
                <p class="text-xs text-ink-400 dark:text-ink-500 mt-2">
                  Your account is linked to your SSO provider. You can log in
                  with either your password or SSO.
                </p>
              {:else}
                {#if $oidcLinkError}
                  <div
                    class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm mb-4"
                  >
                    {$oidcLinkError}
                  </div>
                {/if}
                <p class="text-sm text-ink-500 dark:text-ink-400 mb-4">
                  Link your account to the SSO provider to enable single sign-on
                  login.
                </p>
                <button
                  onclick={handleLinkSso}
                  disabled={linkSsoLoading}
                  class="inline-flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
                >
                  <Link class="w-4 h-4" />
                  {linkSsoLoading ? "Redirecting..." : "Link SSO Account"}
                </button>
              {/if}
            </div>

            <hr class="border-ink-100 dark:border-ink-800" />
          {/if}

          <div>
            <h2
              class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
            >
              <Lock class="w-5 h-5 text-accent-600" />
              Change Password
            </h2>
            <form onsubmit={handlePasswordChange} class="space-y-4">
              <div>
                <label
                  for="current-password"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Current Password
                </label>
                <input
                  id="current-password"
                  type="password"
                  bind:value={currentPassword}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="••••••••"
                  disabled={passwordLoading}
                />
              </div>

              <div>
                <label
                  for="new-password"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  New Password
                </label>
                <input
                  id="new-password"
                  type="password"
                  bind:value={newPassword}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="••••••••"
                  disabled={passwordLoading}
                />
              </div>

              <div>
                <label
                  for="confirm-password"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Confirm New Password
                </label>
                <input
                  id="confirm-password"
                  type="password"
                  bind:value={confirmPassword}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="••••••••"
                  disabled={passwordLoading}
                />
              </div>

              {#if passwordError}
                <div
                  class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
                >
                  {passwordError}
                </div>
              {/if}

              {#if passwordSuccess}
                <div
                  class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
                >
                  Password updated successfully
                </div>
              {/if}

              <button
                type="submit"
                disabled={passwordLoading}
                class="w-full px-4 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
              >
                {passwordLoading ? "Updating..." : "Update Password"}
              </button>
            </form>
          </div>
        </div>
      {/if}

      {#if activeTab === "oidc" && isAdmin}
        <div
          class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
        >
          <div>
            <h2
              class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
            >
              <Shield class="w-5 h-5 text-accent-600" />
              OIDC / Single Sign-On
            </h2>

            <div class="mb-4">
              <div class="flex items-center gap-2 text-sm">
                <span class="text-ink-500">Status:</span>
                {#if oidcStatusLoading}
                  <span class="text-ink-400 dark:text-ink-500"
                    >Checking...</span
                  >
                {:else if oidcConfigured}
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
              Configure an OpenID Connect (OIDC) provider to enable Single
              Sign-On. Users will be able to log in using your identity
              provider.
            </p>

            <form onsubmit={handleOidcSave} class="space-y-4">
              <div>
                <label
                  for="oidc-issuer-url"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Issuer URL
                </label>
                <input
                  id="oidc-issuer-url"
                  type="url"
                  bind:value={oidcIssuerUrl}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="https://auth.example.com"
                  disabled={oidcLoading}
                />
                <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
                  The OIDC provider's issuer URL (must support
                  .well-known/openid-configuration)
                </p>
              </div>

              <div>
                <label
                  for="oidc-client-id"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Client ID
                </label>
                <input
                  id="oidc-client-id"
                  type="text"
                  bind:value={oidcClientId}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="your-client-id"
                  disabled={oidcLoading}
                />
              </div>

              <div>
                <label
                  for="oidc-client-secret"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Client Secret
                </label>
                <input
                  id="oidc-client-secret"
                  type="password"
                  bind:value={oidcClientSecret}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder={oidcConfigured
                    ? "Enter new secret to update"
                    : "Enter your client secret"}
                  disabled={oidcLoading}
                />
                {#if oidcConfigured}
                  <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
                    Leave blank to keep the existing secret
                  </p>
                {/if}
              </div>

              <div>
                <label
                  for="oidc-redirect-uri"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
                >
                  Redirect URI
                </label>
                <input
                  id="oidc-redirect-uri"
                  type="url"
                  bind:value={oidcRedirectUri}
                  class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
                  placeholder="http://localhost:8080/api/auth/oidc/callback"
                  disabled={oidcLoading}
                />
                <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
                  Must match the redirect URI registered with your OIDC provider
                </p>
              </div>

              {#if oidcError}
                <div
                  class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
                >
                  {oidcError}
                </div>
              {/if}

              {#if oidcSuccess}
                <div
                  class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
                >
                  OIDC configuration saved successfully
                </div>
              {/if}

              <button
                type="submit"
                disabled={oidcLoading}
                class="w-full px-4 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
              >
                {oidcLoading
                  ? "Saving..."
                  : oidcConfigured
                    ? "Update Configuration"
                    : "Save Configuration"}
              </button>
            </form>
          </div>
        </div>
      {/if}

      {#if activeTab === "users" && isAdmin}
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
              <p class="text-ink-400 dark:text-ink-400">Loading users...</p>
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
                      class="text-left text-ink-400 dark:text-ink-400 border-b border-ink-100 dark:border-ink-800"
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
                        <td class="py-3 text-ink-500 dark:text-ink-400"
                          >{u.email}</td
                        >
                        <td class="py-3">
                          <span
                            class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium {u.oidc_linked
                              ? 'bg-accent-50 text-accent-700 dark:bg-accent-800/20 dark:text-accent-400'
                              : 'bg-ink-50 text-ink-500 dark:bg-ink-800 dark:text-ink-400'}"
                          >
                            {u.oidc_linked ? "OIDC/SSO" : "Local"}
                          </span>
                        </td>
                        <td class="py-3">
                          {#if u.id === $user?.id}
                            <span
                              class="inline-flex items-center gap-1.5 text-success-700 dark:text-green-400 bg-success-50 dark:bg-green-900/20 px-2.5 py-1 rounded-full text-xs font-medium"
                            >
                              Admin (you)
                            </span>
                          {:else}
                            <button
                              onclick={() => toggleAdmin(u)}
                              class="px-3 py-1 rounded-full text-xs font-medium transition-colors {u.is_admin
                                ? 'bg-success-50 text-success-700 hover:bg-danger-50 hover:text-danger-700 dark:bg-green-900/20 dark:text-green-400 dark:hover:bg-danger-700/10 dark:hover:text-red-400'
                                : 'bg-ink-50 text-ink-500 hover:bg-success-50 hover:text-success-700 dark:bg-ink-800 dark:text-ink-400 dark:hover:bg-green-900/20 dark:hover:text-green-400'}"
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
      {/if}

      {#if activeTab === "preferences"}
        <div
          class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
        >
          <div>
            <h2
              class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
            >
              <Palette class="w-5 h-5 text-accent-600" />
              Display Preferences
            </h2>
            <div class="space-y-6">
              <div>
                <label
                  for="theme-select"
                  class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-3"
                >
                  Theme
                </label>
                <div id="theme-select" class="flex gap-3">
                  {#each themes as t (t)}
                    <button
                      onclick={() => setTheme(t)}
                      class="px-5 py-2.5 rounded-xl font-medium capitalize transition-all {$themePreference ===
                      t
                        ? 'bg-accent-600 text-white shadow-md shadow-accent-600/20'
                        : 'bg-ink-50 text-ink-600 hover:bg-ink-100 dark:bg-ink-800 dark:text-ink-300 dark:hover:bg-ink-700'}"
                    >
                      {t}
                    </button>
                  {/each}
                </div>
                <p class="text-xs text-ink-400 dark:text-ink-500 mt-2">
                  Choose how you prefer biblioteka to appear
                </p>
              </div>

              <hr class="border-ink-100 dark:border-ink-800" />

              <div>
                <label class="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    bind:checked={compactView}
                    class="w-4 h-4 rounded border-ink-300 text-accent-600 focus:ring-2 focus:ring-accent-500"
                  />
                  <div>
                    <span
                      class="block text-sm font-medium text-ink-600 dark:text-ink-300"
                    >
                      Compact View
                    </span>
                    <span class="text-xs text-ink-400 dark:text-ink-500">
                      Display content in a more compact layout
                    </span>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <hr class="border-ink-100 dark:border-ink-800" />

          <div>
            <h2
              class="text-lg font-display font-bold text-ink-900 dark:text-cream-100 mb-4"
            >
              About
            </h2>
            <div class="space-y-3 text-sm text-ink-500 dark:text-ink-400">
              <div class="flex justify-between">
                <span>App Version</span>
                <span class="font-medium text-ink-900 dark:text-cream-100"
                  >1.0.0</span
                >
              </div>
              <div class="flex justify-between">
                <span>Last Updated</span>
                <span class="font-medium text-ink-900 dark:text-cream-100"
                  >Feb 16, 2026</span
                >
              </div>
            </div>
          </div>
        </div>
      {/if}
    </section>
  </div>
</div>
