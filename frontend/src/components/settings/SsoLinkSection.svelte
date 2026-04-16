<script lang="ts">
  import { authStore } from "../../stores/auth.svelte";
  import { createOidcLinkNonce } from "../../lib/api";
  import { Link } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";

  interface Props {
    oidcConfigured: boolean;
  }

  let { oidcConfigured }: Props = $props();
  let linkSsoLoading = $state(false);

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
</script>

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
        Your account is linked to your SSO provider. You can log in with either
        your password or SSO.
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
{/if}
