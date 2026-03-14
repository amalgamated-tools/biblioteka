<script lang="ts">
  import { onMount } from "svelte";
  import { authStore } from "../stores/auth.svelte";
  import { themeStore } from "../stores/theme.svelte";
  import { routerStore } from "../stores/router.svelte";
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
  let userList: AdminUser[] = $state.raw([]);
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
    authStore.oidcLinkError = null;
    try {
      const nonce = await createOidcLinkNonce();
      window.location.href = `/api/auth/oidc/link?nonce=${encodeURIComponent(nonce)}`;
    } catch (err) {
      authStore.oidcLinkError =
        err instanceof Error ? err.message : "Failed to start SSO linking";
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
      userList = userList.map((item) =>
        item.id === u.id ? { ...item, is_admin: !item.is_admin } : item,
      );
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
    <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">
      Settings
    </h1>
    <p class="text-sm text-slate-500 dark:text-slate-400">
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
          class="flex items-center gap-3 px-4 py-3 rounded-lg font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
          'account'
            ? 'bg-blue-50 text-blue-700 border-l-4 border-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'}"
        >
          <Mail class="w-5 h-5" />
          Account
        </button>
        {#if isAdmin}
          <button
            onclick={() => routerStore.navigate("settings/oidc")}
            class="flex items-center gap-3 px-4 py-3 rounded-lg font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
            'oidc'
              ? 'bg-blue-50 text-blue-700 border-l-4 border-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
              : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'}"
          >
            <Shield class="w-5 h-5" />
            OIDC / SSO
          </button>
          <button
            onclick={() => routerStore.navigate("settings/users")}
            class="flex items-center gap-3 px-4 py-3 rounded-lg font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
            'users'
              ? 'bg-blue-50 text-blue-700 border-l-4 border-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
              : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'}"
          >
            <Users class="w-5 h-5" />
            Users
          </button>
        {/if}
        <button
          onclick={() => routerStore.navigate("settings/preferences")}
          class="flex items-center gap-3 px-4 py-3 rounded-lg font-medium whitespace-nowrap sm:whitespace-normal transition-all {activeTab ===
          'preferences'
            ? 'bg-blue-50 text-blue-700 border-l-4 border-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'}"
        >
          <Palette class="w-5 h-5" />
          Preferences
        </button>
      </nav>
    </aside>

    <section class="flex-1">
      {#if activeTab === "account"}
        <div
          class="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 p-6 space-y-6"
        >
          <div>
            <h2
              class="text-xl font-bold text-slate-900 dark:text-slate-100 mb-4 flex items-center gap-2"
            >
              <Mail class="w-5 h-5" />
              Account Information
            </h2>
            <div class="space-y-4">
              <div>
                <label
                  for="email-display"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Email Address
                </label>
                <input
                  id="email-display"
                  type="email"
                  value={authStore.user?.email || ""}
                  disabled
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-slate-50 dark:bg-slate-900 text-slate-600 dark:text-slate-400 cursor-not-allowed"
                />
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  Contact support to change your email address
                </p>
              </div>
            </div>
          </div>

          <hr class="border-slate-200 dark:border-slate-700" />

          {#if oidcConfigured}
            <div>
              <h2
                class="text-xl font-bold text-slate-900 dark:text-slate-100 mb-4 flex items-center gap-2"
              >
                <Link class="w-5 h-5" />
                Single Sign-On
              </h2>
              {#if authStore.user?.oidc_linked}
                <div class="flex items-center gap-2">
                  <span
                    class="inline-flex items-center gap-1.5 text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-900/30 px-2.5 py-1 rounded-full text-sm font-medium"
                  >
                    <span class="w-2 h-2 rounded-full bg-green-500"></span>
                    SSO Connected
                  </span>
                </div>
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-2">
                  Your account is linked to your SSO provider. You can log in
                  with either your password or SSO.
                </p>
              {:else}
                {#if authStore.oidcLinkError}
                  <div
                    class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm mb-4"
                  >
                    {authStore.oidcLinkError}
                  </div>
                {/if}
                <p class="text-sm text-slate-600 dark:text-slate-400 mb-4">
                  Link your account to the SSO provider to enable single sign-on
                  login.
                </p>
                <button
                  onclick={handleLinkSso}
                  disabled={linkSsoLoading}
                  class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
                >
                  <Link class="w-4 h-4" />
                  {linkSsoLoading ? "Redirecting..." : "Link SSO Account"}
                </button>
              {/if}
            </div>

            <hr class="border-slate-200 dark:border-slate-700" />
          {/if}

          <div>
            <h2
              class="text-xl font-bold text-slate-900 dark:text-slate-100 mb-4 flex items-center gap-2"
            >
              <Lock class="w-5 h-5" />
              Change Password
            </h2>
            <form onsubmit={handlePasswordChange} class="space-y-4">
              <div>
                <label
                  for="current-password"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Current Password
                </label>
                <input
                  id="current-password"
                  type="password"
                  bind:value={currentPassword}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder="••••••••"
                  disabled={passwordLoading}
                />
              </div>

              <div>
                <label
                  for="new-password"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  New Password
                </label>
                <input
                  id="new-password"
                  type="password"
                  bind:value={newPassword}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder="••••••••"
                  disabled={passwordLoading}
                />
              </div>

              <div>
                <label
                  for="confirm-password"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Confirm New Password
                </label>
                <input
                  id="confirm-password"
                  type="password"
                  bind:value={confirmPassword}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder="••••••••"
                  disabled={passwordLoading}
                />
              </div>

              {#if passwordError}
                <div
                  class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm"
                >
                  {passwordError}
                </div>
              {/if}

              {#if passwordSuccess}
                <div
                  class="bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 text-green-700 dark:text-green-400 px-4 py-3 rounded-lg text-sm"
                >
                  Password updated successfully
                </div>
              {/if}

              <button
                type="submit"
                disabled={passwordLoading}
                class="w-full px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                {passwordLoading ? "Updating..." : "Update Password"}
              </button>
            </form>
          </div>
        </div>
      {/if}

      {#if activeTab === "oidc" && isAdmin}
        <div
          class="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 p-6 space-y-6"
        >
          <div>
            <h2
              class="text-xl font-bold text-slate-900 dark:text-slate-100 mb-4 flex items-center gap-2"
            >
              <Shield class="w-5 h-5" />
              OIDC / Single Sign-On
            </h2>

            <div class="mb-4">
              <div class="flex items-center gap-2 text-sm">
                <span>Status:</span>
                {#if oidcStatusLoading}
                  <span class="text-slate-500 dark:text-slate-400"
                    >Checking...</span
                  >
                {:else if oidcConfigured}
                  <span
                    class="inline-flex items-center gap-1.5 text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-900/30 px-2.5 py-1 rounded-full font-medium"
                  >
                    <span class="w-2 h-2 rounded-full bg-green-500"></span>
                    Configured
                  </span>
                {:else}
                  <span
                    class="inline-flex items-center gap-1.5 text-slate-700 dark:text-slate-300 bg-slate-50 dark:bg-slate-700 px-2.5 py-1 rounded-full font-medium"
                  >
                    <span class="w-2 h-2 rounded-full bg-slate-400"></span>
                    Not configured
                  </span>
                {/if}
              </div>
            </div>

            <p class="text-sm text-slate-600 dark:text-slate-400 mb-4">
              Configure an OpenID Connect (OIDC) provider to enable Single
              Sign-On. Users will be able to log in using your identity
              provider.
            </p>

            <form onsubmit={handleOidcSave} class="space-y-4">
              <div>
                <label
                  for="oidc-issuer-url"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Issuer URL
                </label>
                <input
                  id="oidc-issuer-url"
                  type="url"
                  bind:value={oidcIssuerUrl}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder="https://auth.example.com"
                  disabled={oidcLoading}
                />
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  The OIDC provider's issuer URL (must support
                  .well-known/openid-configuration)
                </p>
              </div>

              <div>
                <label
                  for="oidc-client-id"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Client ID
                </label>
                <input
                  id="oidc-client-id"
                  type="text"
                  bind:value={oidcClientId}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder="your-client-id"
                  disabled={oidcLoading}
                />
              </div>

              <div>
                <label
                  for="oidc-client-secret"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Client Secret
                </label>
                <input
                  id="oidc-client-secret"
                  type="password"
                  bind:value={oidcClientSecret}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder={oidcConfigured
                    ? "Enter new secret to update"
                    : "Enter your client secret"}
                  disabled={oidcLoading}
                />
                {#if oidcConfigured}
                  <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
                    Leave blank to keep the existing secret
                  </p>
                {/if}
              </div>

              <div>
                <label
                  for="oidc-redirect-uri"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Redirect URI
                </label>
                <input
                  id="oidc-redirect-uri"
                  type="url"
                  bind:value={oidcRedirectUri}
                  class="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none dark:bg-slate-700 dark:text-slate-100"
                  placeholder="http://localhost:8080/api/auth/oidc/callback"
                  disabled={oidcLoading}
                />
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  Must match the redirect URI registered with your OIDC provider
                </p>
              </div>

              {#if oidcError}
                <div
                  class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm"
                >
                  {oidcError}
                </div>
              {/if}

              {#if oidcSuccess}
                <div
                  class="bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 text-green-700 dark:text-green-400 px-4 py-3 rounded-lg text-sm"
                >
                  OIDC configuration saved successfully
                </div>
              {/if}

              <button
                type="submit"
                disabled={oidcLoading}
                class="w-full px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
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
          class="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 p-6 space-y-6"
        >
          <div>
            <h2
              class="text-xl font-bold text-slate-900 dark:text-slate-100 mb-4 flex items-center gap-2"
            >
              <Users class="w-5 h-5" />
              User Management
            </h2>

            {#if usersLoading}
              <p class="text-slate-500 dark:text-slate-400">Loading users...</p>
            {:else if usersError}
              <div
                class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm"
              >
                {usersError}
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-sm">
                  <thead>
                    <tr
                      class="text-left text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-700"
                    >
                      <th class="pb-2 font-medium">Name</th>
                      <th class="pb-2 font-medium">Email</th>
                      <th class="pb-2 font-medium">Type</th>
                      <th class="pb-2 font-medium">Role</th>
                      <th class="pb-2 font-medium">Joined</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each userList as u (u.id)}
                      <tr
                        class="border-b border-slate-100 dark:border-slate-700"
                      >
                        <td class="py-3 text-slate-900 dark:text-slate-100"
                          >{u.name}</td
                        >
                        <td class="py-3 text-slate-600 dark:text-slate-400"
                          >{u.email}</td
                        >
                        <td class="py-3">
                          <span
                            class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium {u.oidc_linked
                              ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                              : 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400'}"
                          >
                            {u.oidc_linked ? "OIDC/SSO" : "Local"}
                          </span>
                        </td>
                        <td class="py-3">
                          {#if u.id === authStore.user?.id}
                            <span
                              class="inline-flex items-center gap-1.5 text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-900/30 px-2.5 py-1 rounded-full text-xs font-medium"
                            >
                              Admin (you)
                            </span>
                          {:else}
                            <button
                              onclick={() => toggleAdmin(u)}
                              class="px-3 py-1 rounded-full text-xs font-medium transition-colors {u.is_admin
                                ? 'bg-green-50 text-green-700 hover:bg-red-50 hover:text-red-700 dark:bg-green-900/30 dark:text-green-400 dark:hover:bg-red-900/30 dark:hover:text-red-400'
                                : 'bg-slate-100 text-slate-600 hover:bg-green-50 hover:text-green-700 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-green-900/30 dark:hover:text-green-400'}"
                            >
                              {u.is_admin ? "Admin" : "User"}
                            </button>
                          {/if}
                        </td>
                        <td class="py-3 text-slate-500 dark:text-slate-400"
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
          class="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 p-6 space-y-6"
        >
          <div>
            <h2
              class="text-xl font-bold text-slate-900 dark:text-slate-100 mb-4 flex items-center gap-2"
            >
              <Palette class="w-5 h-5" />
              Display Preferences
            </h2>
            <div class="space-y-6">
              <div>
                <label
                  for="theme-select"
                  class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3"
                >
                  Theme
                </label>
                <div id="theme-select" class="flex gap-3">
                  {#each themes as t (t)}
                    <button
                      onclick={() => themeStore.set(t)}
                      class="px-4 py-2 rounded-lg font-medium capitalize transition-all {themeStore.preference ===
                      t
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600'}"
                    >
                      {t}
                    </button>
                  {/each}
                </div>
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-2">
                  Choose how you prefer biblioteka to appear
                </p>
              </div>

              <hr class="border-slate-200 dark:border-slate-700" />

              <div>
                <label class="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    bind:checked={compactView}
                    class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500"
                  />
                  <div>
                    <span
                      class="block text-sm font-medium text-slate-700 dark:text-slate-300"
                    >
                      Compact View
                    </span>
                    <span class="text-xs text-slate-500 dark:text-slate-400">
                      Display content in a more compact layout
                    </span>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <hr class="border-slate-200 dark:border-slate-700" />

          <div>
            <h2
              class="text-lg font-bold text-slate-900 dark:text-slate-100 mb-4"
            >
              About
            </h2>
            <div class="space-y-3 text-sm text-slate-600 dark:text-slate-400">
              <div class="flex justify-between">
                <span>App Version</span>
                <span class="font-medium text-slate-900 dark:text-slate-100"
                  >1.0.0</span
                >
              </div>
              <div class="flex justify-between">
                <span>Last Updated</span>
                <span class="font-medium text-slate-900 dark:text-slate-100"
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
