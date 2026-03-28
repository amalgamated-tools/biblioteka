<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    getConfigStatus,
    getSmtpConfig,
    setSmtpConfig,
    testSmtpConfig,
  } from "../../lib/api";
  import { required, validate } from "../../lib/validation";
  import { Mail, Send } from "lucide-svelte";

  interface Props {
    initialSmtpConfigured: boolean;
  }

  let { initialSmtpConfigured }: Props = $props();

  // One-time initialisation – this prop is not expected to change after mount.
  // svelte-ignore state_referenced_locally
  let smtpConfigured = $state(initialSmtpConfigured);
  let smtpEnvOverride = $state(false);
  let smtpPasswordSet = $state(false);
  let smtpHost = $state("");
  let smtpPort = $state("587");
  let smtpUsername = $state("");
  let smtpPassword = $state("");
  let smtpFrom = $state("");
  let smtpTls = $state("starttls");
  let smtpError: string | null = $state(null);
  let smtpSuccessMessage: string | null = $state(null);
  let smtpLoading = $state(false);
  let smtpTestLoading = $state(false);
  let smtpTestMessage: string | null = $state(null);
  let smtpTestError: string | null = $state(null);

  let smtpTestMessageTimeout: ReturnType<typeof setTimeout> | null = null;
  let smtpTestErrorTimeout: ReturnType<typeof setTimeout> | null = null;

  onMount(async () => {
    try {
      const smtp = await getSmtpConfig();
      smtpHost = smtp.host;
      smtpPort = smtp.port || "587";
      smtpUsername = smtp.username;
      smtpFrom = smtp.from;
      smtpTls = smtp.tls || "starttls";
      smtpEnvOverride = smtp.env_override;
      smtpPasswordSet = smtp.password_set;
    } catch {
      // ignore - user can re-enter
    }
  });

  onDestroy(() => {
    if (smtpTestMessageTimeout !== null) {
      clearTimeout(smtpTestMessageTimeout);
      smtpTestMessageTimeout = null;
    }
    if (smtpTestErrorTimeout !== null) {
      clearTimeout(smtpTestErrorTimeout);
      smtpTestErrorTimeout = null;
    }
  });

  async function handleSmtpSave(e: SubmitEvent) {
    e.preventDefault();
    smtpError = null;
    smtpSuccessMessage = null;

    smtpError =
      validate(smtpHost, [required("SMTP Host is required")]) ??
      validate(smtpFrom, [required("From Address is required")]);
    if (smtpError) return;

    if (
      !smtpPassword.trim() &&
      smtpUsername.trim() &&
      (!smtpPasswordSet || smtpEnvOverride)
    ) {
      smtpError = "Password is required when username is set";
      return;
    }

    smtpLoading = true;

    try {
      const result = await setSmtpConfig({
        host: smtpHost.trim(),
        port: smtpPort.trim() || "587",
        username: smtpUsername.trim(),
        password: smtpPassword.trim(),
        from: smtpFrom.trim(),
        tls: smtpTls,
      });
      const status = await getConfigStatus();
      smtpSuccessMessage = result.message;
      smtpConfigured = status.smtp_configured;
      if (smtpPassword.trim()) {
        smtpPasswordSet = true;
      } else if (!smtpUsername.trim()) {
        // Saved with no username → credentials were cleared
        smtpPasswordSet = false;
      }
      smtpPassword = "";
    } catch (err) {
      smtpError =
        err instanceof Error
          ? err.message
          : "Failed to save SMTP configuration";
    } finally {
      smtpLoading = false;
    }
  }

  async function handleSmtpTest() {
    smtpTestMessage = null;
    smtpTestError = null;
    smtpTestLoading = true;

    try {
      const result = await testSmtpConfig();
      smtpTestMessage = result.message;
      if (smtpTestMessageTimeout !== null) {
        clearTimeout(smtpTestMessageTimeout);
        smtpTestMessageTimeout = null;
      }
      smtpTestMessageTimeout = setTimeout(() => {
        smtpTestMessage = null;
        smtpTestMessageTimeout = null;
      }, 5000);
    } catch (err) {
      smtpTestError =
        err instanceof Error ? err.message : "Failed to send test email";
      if (smtpTestErrorTimeout !== null) {
        clearTimeout(smtpTestErrorTimeout);
        smtpTestErrorTimeout = null;
      }
      smtpTestErrorTimeout = setTimeout(() => {
        smtpTestError = null;
        smtpTestErrorTimeout = null;
      }, 5000);
    } finally {
      smtpTestLoading = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 space-y-6 animate-fade-in"
>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h2
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 flex items-center gap-2"
      >
        <Send class="w-5 h-5 text-accent-600" />
        Email / SMTP Configuration
      </h2>
      {#if smtpConfigured}
        <button
          onclick={handleSmtpTest}
          disabled={smtpTestLoading}
          class="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-accent-600 to-accent-700 text-white text-sm font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
        >
          <Mail class="w-4 h-4" />
          {smtpTestLoading ? "Sending..." : "Send Test Email"}
        </button>
      {/if}
    </div>

    {#if smtpTestMessage}
      <div
        role="status"
        class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm animate-scale-in mb-4"
      >
        {smtpTestMessage}
      </div>
    {/if}

    {#if smtpTestError}
      <div
        role="alert"
        class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in mb-4"
      >
        {smtpTestError}
      </div>
    {/if}

    <div class="mb-4">
      <div class="flex items-center gap-2 text-sm">
        <span class="text-ink-500">Status:</span>
        {#if smtpConfigured}
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
      Configure SMTP settings to enable email notifications from Biblioteka.
    </p>

    {#if smtpEnvOverride}
      <div
        class="bg-accent-50 dark:bg-accent-800/20 border border-accent-200 dark:border-accent-700/30 text-accent-700 dark:text-accent-400 px-4 py-3 rounded-xl text-sm mb-4"
      >
        SMTP is configured via environment variables and cannot be changed here.
        Remove the SMTP environment variables from the server to manage settings
        through this UI.
      </div>
    {/if}

    <form onsubmit={handleSmtpSave} class="space-y-4">
      <div>
        <label
          for="smtp-host"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          SMTP Host
        </label>
        <input
          id="smtp-host"
          type="text"
          bind:value={smtpHost}
          class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          placeholder="smtp.example.com"
          disabled={smtpLoading || smtpEnvOverride}
        />
      </div>

      <div>
        <label
          for="smtp-port"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Port
        </label>
        <input
          id="smtp-port"
          type="text"
          bind:value={smtpPort}
          class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          placeholder="587"
          disabled={smtpLoading || smtpEnvOverride}
        />
      </div>

      <div>
        <label
          for="smtp-username"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Username
        </label>
        <input
          id="smtp-username"
          type="text"
          bind:value={smtpUsername}
          class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          placeholder="user@example.com"
          disabled={smtpLoading || smtpEnvOverride}
        />
      </div>

      {#if !smtpEnvOverride}
        <div>
          <label
            for="smtp-password"
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
          >
            Password
          </label>
          <input
            id="smtp-password"
            type="password"
            bind:value={smtpPassword}
            class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
            placeholder={smtpPasswordSet
              ? "Enter new password to update"
              : "Enter your SMTP password"}
            disabled={smtpLoading}
          />
          {#if smtpPasswordSet}
            <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
              Leave blank to keep the existing password
            </p>
          {/if}
        </div>
      {/if}

      <div>
        <label
          for="smtp-from"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          From Address
        </label>
        <input
          id="smtp-from"
          type="email"
          bind:value={smtpFrom}
          class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          placeholder="noreply@example.com"
          disabled={smtpLoading || smtpEnvOverride}
        />
        <p class="text-xs text-ink-400 dark:text-ink-500 mt-1">
          The email address that outgoing messages will be sent from
        </p>
      </div>

      <div>
        <label
          for="smtp-tls"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          TLS Mode
        </label>
        <select
          id="smtp-tls"
          bind:value={smtpTls}
          class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          disabled={smtpLoading || smtpEnvOverride}
        >
          <option value="starttls">STARTTLS</option>
          <option value="tls">TLS</option>
          <option value="none">None</option>
        </select>
      </div>

      {#if smtpError}
        <div
          role="alert"
          class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
        >
          {smtpError}
        </div>
      {/if}

      {#if smtpSuccessMessage}
        <div
          role="status"
          class="bg-success-50 dark:bg-green-900/20 border border-success-600/20 dark:border-green-700/30 text-success-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm animate-scale-in"
        >
          {smtpSuccessMessage}
        </div>
      {/if}

      {#if !smtpEnvOverride}
        <button
          type="submit"
          disabled={smtpLoading}
          class="w-full px-4 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white font-semibold rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all disabled:opacity-50 shadow-md shadow-accent-600/20"
        >
          {smtpLoading
            ? "Saving..."
            : smtpConfigured
              ? "Update Configuration"
              : "Save Configuration"}
        </button>
      {/if}
    </form>
  </div>
</div>
