<script lang="ts">
  import { onMount } from "svelte";
  import { authStore } from "../../stores/auth.svelte";
  import {
    changePassword,
    createOidcLinkNonce,
    updateProfile,
    getPasskeyEnabled,
    listPasskeyCredentials,
    deletePasskeyCredential,
    beginPasskeyRegistration,
    finishPasskeyRegistration,
  } from "../../lib/api";
  import type { PasskeyCredential } from "../../types";
  import { required, minLength, matches, validate } from "../../lib/validation";
  import { AutoDismissTimer } from "../../lib/autoDismissTimer.svelte";
  import { Lock, Mail, Link, User, KeyRound, Trash2 } from "lucide-svelte";
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

  // Keep timer cleanup in a separate $effect that reads no reactive state so it is registered
  // once on mount and only runs on destroy. This avoids re-registering or cleaning up the
  // timers when the displayName-sync $effect reruns.
  $effect(() => {
    return () => {
      successTimer.clear();
      nameSuccessTimer.clear();
      passkeySuccessTimer.clear();
    };
  });

  // Passkeys
  let passkeyEnabled = $state(false);
  let passkeys = $state<PasskeyCredential[]>([]);
  let passkeyError: string | null = $state(null);
  let passkeyRegisterName = $state("");
  let passkeyRegistering = $state(false);
  let passkeyDeleting = $state<string | null>(null);
  const passkeySuccessTimer = new AutoDismissTimer();

  onMount(async () => {
    try {
      passkeyEnabled = await getPasskeyEnabled();
      if (passkeyEnabled) {
        passkeys = await listPasskeyCredentials();
      }
    } catch {
      // Passkey availability check failed; silently treat as disabled.
    }
  });

  async function handleRegisterPasskey(e: SubmitEvent) {
    e.preventDefault();
    passkeyError = null;

    const trimmedName = passkeyRegisterName.trim();
    if (!trimmedName) {
      passkeyError = "Passkey name is required";
      return;
    }

    passkeyRegistering = true;
    try {
      const { session_id, options } =
        await beginPasskeyRegistration(trimmedName);

      const credential = await navigator.credentials.create({
        publicKey: (options as { publicKey: unknown })
          .publicKey as PublicKeyCredentialCreationOptions,
      });

      if (!credential || !(credential instanceof PublicKeyCredential)) {
        passkeyError = "No passkey was created";
        return;
      }

      const credentialJSON = (
        credential as PublicKeyCredential & { toJSON(): unknown }
      ).toJSON();
      const stored = await finishPasskeyRegistration(
        session_id,
        credentialJSON,
      );

      passkeys = [stored, ...passkeys];
      passkeyRegisterName = "";
      passkeySuccessTimer.show();
    } catch (err) {
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        // User cancelled — don't show an error.
      } else {
        passkeyError =
          err instanceof Error ? err.message : "Failed to register passkey";
      }
    } finally {
      passkeyRegistering = false;
    }
  }

  async function handleDeletePasskey(id: string) {
    passkeyError = null;
    passkeyDeleting = id;
    try {
      await deletePasskeyCredential(id);
      passkeys = passkeys.filter((p) => p.id !== id);
    } catch (err) {
      passkeyError =
        err instanceof Error ? err.message : "Failed to delete passkey";
    } finally {
      passkeyDeleting = null;
    }
  }

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

  {#if passkeyEnabled}
    <hr class="border-ink-100 dark:border-ink-800" />

    <div>
      <h2
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
      >
        <KeyRound class="w-5 h-5 text-accent-600" aria-hidden="true" />
        Passkeys
      </h2>

      <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
        Passkeys let you sign in with biometrics or a hardware security key
        instead of a password.
      </p>

      {#if passkeys.length > 0}
        <ul class="space-y-2 mb-4" aria-label="Registered passkeys">
          {#each passkeys as passkey (passkey.id)}
            <li
              class="flex items-center justify-between gap-2 p-3 rounded-xl border border-ink-100 dark:border-ink-700 bg-cream-50 dark:bg-ink-800"
            >
              <div class="flex items-center gap-2 min-w-0">
                <KeyRound
                  class="w-4 h-4 shrink-0 text-accent-600"
                  aria-hidden="true"
                />
                <span
                  class="text-sm font-medium text-ink-800 dark:text-cream-100 truncate"
                  >{passkey.name}</span
                >
                <span class="text-xs text-ink-400 dark:text-ink-400 shrink-0">
                  {new Date(passkey.created_at).toLocaleDateString()}
                </span>
              </div>
              <button
                type="button"
                aria-label="Delete passkey {passkey.name}"
                disabled={passkeyDeleting === passkey.id}
                onclick={() => handleDeletePasskey(passkey.id)}
                class="p-1.5 rounded-lg text-ink-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-40"
              >
                <Trash2 class="w-4 h-4" aria-hidden="true" />
              </button>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="text-sm text-ink-400 dark:text-ink-400 mb-4">
          No passkeys registered yet.
        </p>
      {/if}

      <form
        onsubmit={handleRegisterPasskey}
        aria-label="Register new passkey"
        class="flex gap-2"
      >
        <TextInput
          id="passkey-name"
          type="text"
          bind:value={passkeyRegisterName}
          placeholder="Name (e.g. My iPhone)"
          disabled={passkeyRegistering}
          class="flex-1 py-2.5"
          aria-label="Passkey name"
        />
        <Button
          type="submit"
          disabled={passkeyRegistering}
          class="px-4 py-2.5 shrink-0"
        >
          {passkeyRegistering ? "Registering…" : "Add Passkey"}
        </Button>
      </form>

      {#if passkeyError}
        <AlertBanner variant="error" class="mt-3">{passkeyError}</AlertBanner>
      {/if}

      {#if passkeySuccessTimer.visible}
        <AlertBanner variant="success" class="mt-3"
          >Passkey registered successfully</AlertBanner
        >
      {/if}
    </div>
  {/if}
</div>
