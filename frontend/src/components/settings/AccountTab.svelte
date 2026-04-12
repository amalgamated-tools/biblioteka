<script lang="ts">
  import { authStore } from "../../stores/auth.svelte";
  import {
    changePassword,
    createOidcLinkNonce,
    updateProfile,
  } from "../../lib/api";
  import { required, minLength, matches, validate } from "../../lib/validation";
  import { AutoDismissTimer } from "../../lib/autoDismissTimer.svelte";
  import { onDestroy } from "svelte";
  import { Lock, Mail, Link, User } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    oidcConfigured: boolean;
  }

  let { oidcConfigured }: Props = $props();

  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let passwordError: string | null = $state(null);
  let passwordLoading = $state(false);
  let linkSsoLoading = $state(false);
  const successTimer = new AutoDismissTimer();

  let displayName = $state(authStore.user?.name ?? "");
  let nameError: string | null = $state(null);
  let nameLoading = $state(false);
  const nameSuccessTimer = new AutoDismissTimer();

  // Keep displayName in sync when the auth store user loads or updates externally.
  $effect(() => {
    if (authStore.user && !nameLoading) {
      displayName = authStore.user.name;
    }
  });

  onDestroy(() => {
    successTimer.clear();
    nameSuccessTimer.clear();
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

  async function handleNameUpdate(e: SubmitEvent) {
    e.preventDefault();
    nameError = null;
    nameSuccessTimer.clear();

    nameError = validate(displayName, [required("Display name is required")]);
    if (nameError) return;

    nameLoading = true;
    try {
      const updated = await updateProfile(displayName);
      if (authStore.user) {
        authStore.user = { ...authStore.user, name: updated.name };
      }
      nameSuccessTimer.show();
    } catch (err) {
      nameError =
        err instanceof Error ? err.message : "Failed to update display name";
    } finally {
      nameLoading = false;
    }
  }

  async function handlePasswordChange(e: SubmitEvent) {
    e.preventDefault();
    passwordError = null;
    successTimer.clear();

    passwordError =
      validate(currentPassword, [required("Current password is required")]) ??
      validate(newPassword, [
        required("New password is required"),
        minLength(6, "New password must be at least 6 characters"),
      ]) ??
      validate(confirmPassword, [
        matches(newPassword, "Passwords do not match"),
      ]);
    if (passwordError) return;

    passwordLoading = true;

    try {
      await changePassword(currentPassword, newPassword);
      successTimer.show();
      currentPassword = "";
      newPassword = "";
      confirmPassword = "";
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
      <Mail class="w-5 h-5 text-accent-600" aria-hidden="true" />
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
        <TextInput
          id="email-display"
          type="email"
          value={authStore.user?.email || ""}
          disabled
          class="w-full py-2.5"
          aria-describedby="email-display-hint"
        />
        <p
          id="email-display-hint"
          class="text-xs text-ink-500 dark:text-ink-300 mt-1"
        >
          Contact support to change your email address
        </p>
      </div>
    </div>
  </div>

  <hr class="border-ink-100 dark:border-ink-800" />

  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
    >
      <User class="w-5 h-5 text-accent-600" aria-hidden="true" />
      Display Name
    </h2>
    <form
      onsubmit={handleNameUpdate}
      aria-label="Update display name"
      class="space-y-4"
    >
      <div>
        <label
          for="display-name"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Name
        </label>
        <TextInput
          id="display-name"
          type="text"
          bind:value={displayName}
          autocomplete="name"
          class="w-full py-2.5"
          placeholder="Your name"
          disabled={nameLoading}
        />
      </div>

      {#if nameError}
        <AlertBanner variant="error">{nameError}</AlertBanner>
      {/if}

      {#if nameSuccessTimer.visible}
        <AlertBanner variant="success">Display name updated</AlertBanner>
      {/if}

      <Button type="submit" disabled={nameLoading} class="w-full px-4 py-2.5">
        {nameLoading ? "Saving..." : "Save Name"}
      </Button>
    </form>
  </div>

  <hr class="border-ink-100 dark:border-ink-800" />

  {#if oidcConfigured}
    <div>
      <h2
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
      >
        <Link class="w-5 h-5 text-accent-600" aria-hidden="true" />
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
        <p class="text-xs text-ink-500 dark:text-ink-300 mt-2">
          Your account is linked to your SSO provider. You can log in with
          either your password or SSO.
        </p>
      {:else}
        {#if authStore.oidcLinkError}
          <AlertBanner variant="error" class="mb-4"
            >{authStore.oidcLinkError}</AlertBanner
          >
        {/if}
        <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
          Link your account to the SSO provider to enable single sign-on login.
        </p>
        <Button
          onclick={handleLinkSso}
          disabled={linkSsoLoading}
          class="inline-flex items-center gap-2 px-4 py-2.5"
        >
          <Link class="w-4 h-4" aria-hidden="true" />
          {linkSsoLoading ? "Redirecting..." : "Link SSO Account"}
        </Button>
      {/if}
    </div>

    <hr class="border-ink-100 dark:border-ink-800" />
  {/if}

  <div>
    <h2
      class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
    >
      <Lock class="w-5 h-5 text-accent-600" aria-hidden="true" />
      Change Password
    </h2>
    <form
      onsubmit={handlePasswordChange}
      aria-label="Change password"
      class="space-y-4"
    >
      <div>
        <label
          for="current-password"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Current Password
        </label>
        <TextInput
          id="current-password"
          type="password"
          bind:value={currentPassword}
          autocomplete="current-password"
          class="w-full py-2.5"
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
        <TextInput
          id="new-password"
          type="password"
          bind:value={newPassword}
          autocomplete="new-password"
          class="w-full py-2.5"
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
        <TextInput
          id="confirm-password"
          type="password"
          bind:value={confirmPassword}
          autocomplete="new-password"
          class="w-full py-2.5"
          placeholder="••••••••"
          disabled={passwordLoading}
        />
      </div>

      {#if passwordError}
        <AlertBanner variant="error">{passwordError}</AlertBanner>
      {/if}

      {#if successTimer.visible}
        <AlertBanner variant="success"
          >Password updated successfully</AlertBanner
        >
      {/if}

      <Button
        type="submit"
        disabled={passwordLoading}
        class="w-full px-4 py-2.5"
      >
        {passwordLoading ? "Updating..." : "Update Password"}
      </Button>
    </form>
  </div>
</div>
