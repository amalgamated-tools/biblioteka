<script lang="ts">
  import { onDestroy } from "svelte";
  import { setOidcConfig } from "../../lib/api";
  import { required, validate } from "../../lib/validation";
  import { Shield } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    initialOidcConfigured: boolean;
    initialIssuerUrl?: string;
    initialClientId?: string;
    initialRedirectUri?: string;
    onOidcSaved: (config: {
      configured: boolean;
      issuerUrl: string;
      clientId: string;
      redirectUri: string;
    }) => void;
  }

  let {
    initialOidcConfigured,
    initialIssuerUrl = "",
    initialClientId = "",
    initialRedirectUri = "",
    onOidcSaved,
  }: Props = $props();

  // One-time initialisation – these props are not expected to change after mount.
  // svelte-ignore state_referenced_locally
  let oidcConfigured = $state(initialOidcConfigured);
  // svelte-ignore state_referenced_locally
  let oidcIssuerUrl = $state(initialIssuerUrl);
  // svelte-ignore state_referenced_locally
  let oidcClientId = $state(initialClientId);
  let oidcClientSecret = $state("");
  // svelte-ignore state_referenced_locally
  let oidcRedirectUri = $state(initialRedirectUri);
  let oidcError: string | null = $state(null);
  let oidcSuccess = $state(false);
  let oidcLoading = $state(false);
  let successTimer: ReturnType<typeof setTimeout> | undefined;

  onDestroy(() => {
    if (successTimer) clearTimeout(successTimer);
  });

  async function handleOidcSave(e: SubmitEvent) {
    e.preventDefault();
    oidcError = null;
    oidcSuccess = false;

    oidcError =
      validate(oidcIssuerUrl, [required("Issuer URL is required")]) ??
      validate(oidcClientId, [required("Client ID is required")]) ??
      (!oidcConfigured
        ? validate(oidcClientSecret, [required("Client Secret is required")])
        : null) ??
      validate(oidcRedirectUri, [required("Redirect URI is required")]);
    if (oidcError) return;

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
      onOidcSaved({
        configured: true,
        issuerUrl: oidcIssuerUrl.trim(),
        clientId: oidcClientId.trim(),
        redirectUri: oidcRedirectUri.trim(),
      });
      successTimer = setTimeout(() => (oidcSuccess = false), 3000);
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
        {#if oidcConfigured}
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
      Configure an OpenID Connect (OIDC) provider to enable Single Sign-On.
      Users will be able to log in using your identity provider.
    </p>

    <form onsubmit={handleOidcSave} class="space-y-4">
      <div>
        <label
          for="oidc-issuer-url"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Issuer URL
        </label>
        <TextInput
          id="oidc-issuer-url"
          type="url"
          bind:value={oidcIssuerUrl}
          class="w-full py-2.5"
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
        <TextInput
          id="oidc-client-id"
          type="text"
          bind:value={oidcClientId}
          class="w-full py-2.5"
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
        <TextInput
          id="oidc-client-secret"
          type="password"
          bind:value={oidcClientSecret}
          class="w-full py-2.5"
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
        <TextInput
          id="oidc-redirect-uri"
          type="url"
          bind:value={oidcRedirectUri}
          class="w-full py-2.5"
          placeholder="http://localhost:8080/api/auth/oidc/callback"
          disabled={oidcLoading}
        />
        <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
          Must match the redirect URI registered with your OIDC provider
        </p>
      </div>

      {#if oidcError}
        <AlertBanner variant="error">{oidcError}</AlertBanner>
      {/if}

      {#if oidcSuccess}
        <AlertBanner variant="success"
          >OIDC configuration saved successfully</AlertBanner
        >
      {/if}

      <Button
        type="submit"
        disabled={oidcLoading}
        class="w-full px-4 py-2.5"
      >
        {oidcLoading
          ? "Saving..."
          : oidcConfigured
            ? "Update Configuration"
            : "Save Configuration"}
      </Button>
    </form>
  </div>
</div>
