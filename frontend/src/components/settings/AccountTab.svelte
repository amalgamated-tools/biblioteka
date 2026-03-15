<script lang="ts">
  import { onDestroy } from "svelte";
  import { authStore } from "../../stores/auth.svelte";
  import {
    changePassword,
    createOidcLinkNonce,
  } from "../../lib/api";
  import { Lock, Mail, Link } from "lucide-svelte";

  interface Props {
    oidcConfigured: boolean;
  }

  let { oidcConfigured }: Props = $props();

  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let passwordError: string | null = $state(null);
  let passwordSuccess = $state(false);
  let passwordLoading = $state(false);
  let linkSsoLoading = $state(false);
  let successTimer: ReturnType<typeof setTimeout> | undefined;

  onDestroy(() => {
    if (successTimer) clearTimeout(successTimer);
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
      successTimer = setTimeout(() => (passwordSuccess = false), 3000);
    } catch (err) {
      passwordError =
        err instanceof Error ? err.message : "Failed to update password";
    } finally {
      passwordLoading = false;
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
          value={authStore.user?.email || ""}
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
      {#if authStore.user?.oidc_linked}
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
        {#if authStore.oidcLinkError}
          <div
            class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm mb-4"
          >
            {authStore.oidcLinkError}
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
