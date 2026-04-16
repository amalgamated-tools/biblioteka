<script lang="ts">
  import { onMount } from "svelte";
  import {
    getPasskeyEnabled,
    listPasskeyCredentials,
    deletePasskeyCredential,
    prepareCreationOptions,
    beginPasskeyRegistration,
    finishPasskeyRegistration,
  } from "../../lib/api";
  import type { PasskeyCredential } from "../../types";
  import { AutoDismissTimer } from "../../lib/autoDismissTimer.svelte";
  import { KeyRound } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import PasskeyList from "./PasskeyList.svelte";
  import TextInput from "../ui/TextInput.svelte";

  let passkeyEnabled = $state(false);
  let passkeys = $state<PasskeyCredential[]>([]);
  let passkeyError: string | null = $state(null);
  let passkeyRegisterName = $state("");
  let passkeyRegistering = $state(false);
  let passkeyDeleting = $state<string | null>(null);
  const passkeySuccessTimer = new AutoDismissTimer();

  $effect(() => {
    return () => {
      passkeySuccessTimer.clear();
    };
  });

  onMount(async () => {
    try {
      passkeyEnabled = await getPasskeyEnabled();
      if (passkeyEnabled) {
        passkeys = await listPasskeyCredentials();
      }
    } catch {
      // Ignore passkey initialization failures and keep the current state.
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
        publicKey: prepareCreationOptions(options),
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
        // User canceled — don't show an error.
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
</script>

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

    <PasskeyList {passkeys} {passkeyDeleting} onDelete={handleDeletePasskey} />

    <form
      onsubmit={handleRegisterPasskey}
      aria-label="Register new passkey"
      class="space-y-2"
    >
      <label
        for="passkey-name"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300"
      >
        Passkey name
      </label>
      <div class="flex gap-2">
        <TextInput
          id="passkey-name"
          type="text"
          bind:value={passkeyRegisterName}
          placeholder="e.g. My iPhone"
          disabled={passkeyRegistering}
          class="flex-1 py-2.5"
        />
        <Button
          type="submit"
          disabled={passkeyRegistering}
          class="px-4 py-2.5 shrink-0"
        >
          {passkeyRegistering ? "Registering…" : "Add Passkey"}
        </Button>
      </div>
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
