<script lang="ts">
  import AccountInfoSection from "./AccountInfoSection.svelte";
  import ChangePasswordSection from "./ChangePasswordSection.svelte";
  import DisplayNameSection from "./DisplayNameSection.svelte";
  import PasskeysSection from "./PasskeysSection.svelte";
  import SsoLinkSection from "./SsoLinkSection.svelte";

  interface Props {
    oidcConfigured: boolean;
  }

  let { oidcConfigured }: Props = $props();
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <AccountInfoSection />

  <hr class="border-ink-100 dark:border-ink-800" />

  <DisplayNameSection />

  <hr class="border-ink-100 dark:border-ink-800" />

  <SsoLinkSection {oidcConfigured} />

  {#if oidcConfigured}
    <hr class="border-ink-100 dark:border-ink-800" />
  {/if}

  <ChangePasswordSection />

  <PasskeysSection />
</div>
