<script lang="ts">
  import { onDestroy } from "svelte";
  import { setOidcConfig } from "../../lib/api";
  import { Shield } from "lucide-svelte";

  interface Props {
    initialOidcConfigured: boolean;
    initialIssuerUrl?: string;
    initialClientId?: string;
    initialRedirectUri?: string;
    onOidcSaved: (config: { configured: boolean; issuerUrl: string; clientId: string; redirectUri: string }) => void;
  }

  let {
    initialOidcConfigured,
    initialIssuerUrl = "",
    initialClientId = "",
    initialRedirectUri = "",
    onOidcSaved,
  }: Props = $props();

  let oidcConfigured = $state(initialOidcConfigured);
  let oidcIssuerUrl = $state(initialIssuerUrl);
  let oidcClientId = $state(initialClientId);
  let oidcClientSecret = $state("");
  let oidcRedirectUri = $state(initialRedirectUri);
  let oidcError: string | null = $state(null);
  let oidcSuccess = $state(false);
  let oidcLoading = $state(false);
  let successTimer: ReturnType<typeof setTimeout> | undefined;

  let saveButtonLabel = $derived.by(() => {
    if (oidcLoading) return "Saving...";
    if (oidcConfigured) return "Update Configuration";
    return "Save Configuration";
  });

  onDestroy(() => {
    if (successTimer) clearTimeout(successTimer);
  });

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
        {saveButtonLabel}
      </button>
    </form>
  </div>
</div>
