<script lang="ts">
  import {
    getConfigStatus,
    getSmtpConfig,
    setSmtpConfig,
    testSmtpConfig,
  } from "../../lib/api";
  import { required, validate } from "../../lib/validation";
  import { AutoDismissTimer } from "../../lib/autoDismissTimer.svelte";
  import { Mail, Send } from "lucide-svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";

  interface Props {
    initialSmtpConfigured: boolean;
  }

  interface SmtpStatus {
    configured: boolean;
    envOverride: boolean;
    passwordSet: boolean;
    error: string | null;
    successMessage: string | null;
    loading: boolean;
    testLoading: boolean;
    testMessage: string | null;
    testError: string | null;
  }

  let { initialSmtpConfigured }: Props = $props();

  let smtpForm = $state({
    host: "",
    port: "587",
    username: "",
    password: "",
    from: "",
    tls: "starttls",
  });

  // One-time initialisation – this prop is not expected to change after mount.
  // svelte-ignore state_referenced_locally
  let smtpStatus: SmtpStatus = $state({
    configured: initialSmtpConfigured,
    envOverride: false,
    passwordSet: false,
    error: null,
    successMessage: null,
    loading: false,
    testLoading: false,
    testMessage: null,
    testError: null,
  });

  const testMessageTimer = new AutoDismissTimer(5000);
  const testErrorTimer = new AutoDismissTimer(5000);

  let smtpSubmitLabel = $derived.by(() => {
    if (smtpStatus.loading) return "Saving...";
    return smtpStatus.configured
      ? "Update Configuration"
      : "Save Configuration";
  });

  $effect(() => {
    void (async () => {
      try {
        const smtp = await getSmtpConfig();
        smtpForm.host = smtp.host;
        smtpForm.port = smtp.port || "587";
        smtpForm.username = smtp.username;
        smtpForm.from = smtp.from;
        smtpForm.tls = smtp.tls || "starttls";
        smtpStatus.envOverride = smtp.env_override;
        smtpStatus.passwordSet = smtp.password_set;
      } catch {
        // ignore - user can re-enter
      }
    })();
    return () => {
      testMessageTimer.clear();
      testErrorTimer.clear();
    };
  });

  async function handleSmtpSave(e: SubmitEvent) {
    e.preventDefault();
    smtpStatus.error = null;
    smtpStatus.successMessage = null;

    smtpStatus.error =
      validate(smtpForm.host, [required("SMTP Host is required")]) ??
      validate(smtpForm.from, [required("From Address is required")]);
    if (smtpStatus.error) return;

    if (
      !smtpForm.password.trim() &&
      smtpForm.username.trim() &&
      (!smtpStatus.passwordSet || smtpStatus.envOverride)
    ) {
      smtpStatus.error = "Password is required when username is set";
      return;
    }

    smtpStatus.loading = true;

    try {
      const result = await setSmtpConfig({
        host: smtpForm.host.trim(),
        port: smtpForm.port.trim() || "587",
        username: smtpForm.username.trim(),
        password: smtpForm.password.trim(),
        from: smtpForm.from.trim(),
        tls: smtpForm.tls,
      });
      const status = await getConfigStatus();
      smtpStatus.successMessage = result.message;
      smtpStatus.configured = status.smtp_configured;
      if (smtpForm.password.trim()) {
        smtpStatus.passwordSet = true;
      } else if (!smtpForm.username.trim()) {
        // Saved with no username → credentials were cleared
        smtpStatus.passwordSet = false;
      }
      smtpForm.password = "";
    } catch (err) {
      smtpStatus.error =
        err instanceof Error
          ? err.message
          : "Failed to save SMTP configuration";
    } finally {
      smtpStatus.loading = false;
    }
  }

  async function handleSmtpTest() {
    testMessageTimer.clear();
    testErrorTimer.clear();
    smtpStatus.testMessage = null;
    smtpStatus.testError = null;
    smtpStatus.testLoading = true;

    try {
      const result = await testSmtpConfig();
      smtpStatus.testMessage = result.message;
      testMessageTimer.show();
    } catch (err) {
      smtpStatus.testError =
        err instanceof Error ? err.message : "Failed to send test email";
      testErrorTimer.show();
    } finally {
      smtpStatus.testLoading = false;
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
        <Send class="w-5 h-5 text-accent-600" aria-hidden="true" />
        Email / SMTP Configuration
      </h2>
      {#if smtpStatus.configured}
        <Button
          onclick={handleSmtpTest}
          disabled={smtpStatus.testLoading}
          class="inline-flex items-center gap-2 px-4 py-2 text-sm"
        >
          <Mail class="w-4 h-4" aria-hidden="true" />
          {smtpStatus.testLoading ? "Sending..." : "Send Test Email"}
        </Button>
      {/if}
    </div>

    {#if testMessageTimer.visible && smtpStatus.testMessage}
      <AlertBanner variant="success" class="mb-4"
        >{smtpStatus.testMessage}</AlertBanner
      >
    {/if}

    {#if testErrorTimer.visible && smtpStatus.testError}
      <AlertBanner variant="error" class="mb-4"
        >{smtpStatus.testError}</AlertBanner
      >
    {/if}

    <div class="mb-4">
      <div class="flex items-center gap-2 text-sm">
        <span class="text-ink-500 dark:text-ink-300">Status:</span>
        {#if smtpStatus.configured}
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

    <p class="text-sm text-ink-500 dark:text-ink-300 mb-4">
      Configure SMTP settings to enable email notifications from Biblioteka.
    </p>

    {#if smtpStatus.envOverride}
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
        <TextInput
          id="smtp-host"
          type="text"
          bind:value={smtpForm.host}
          class="w-full py-2.5"
          placeholder="smtp.example.com"
          disabled={smtpStatus.loading || smtpStatus.envOverride}
        />
      </div>

      <div>
        <label
          for="smtp-port"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Port
        </label>
        <TextInput
          id="smtp-port"
          type="text"
          bind:value={smtpForm.port}
          class="w-full py-2.5"
          placeholder="587"
          disabled={smtpStatus.loading || smtpStatus.envOverride}
        />
      </div>

      <div>
        <label
          for="smtp-username"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
        >
          Username
        </label>
        <TextInput
          id="smtp-username"
          type="text"
          bind:value={smtpForm.username}
          class="w-full py-2.5"
          placeholder="user@example.com"
          disabled={smtpStatus.loading || smtpStatus.envOverride}
        />
      </div>

      {#if !smtpStatus.envOverride}
        <div>
          <label
            for="smtp-password"
            class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
          >
            Password
          </label>
          <TextInput
            id="smtp-password"
            type="password"
            bind:value={smtpForm.password}
            class="w-full py-2.5"
            placeholder={smtpStatus.passwordSet
              ? "Enter new password to update"
              : "Enter your SMTP password"}
            disabled={smtpStatus.loading}
            aria-describedby={smtpStatus.passwordSet
              ? "smtp-password-hint"
              : undefined}
          />
          {#if smtpStatus.passwordSet}
            <p
              id="smtp-password-hint"
              class="text-xs text-ink-500 dark:text-ink-300 mt-1"
            >
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
        <TextInput
          id="smtp-from"
          type="email"
          bind:value={smtpForm.from}
          class="w-full py-2.5"
          placeholder="noreply@example.com"
          disabled={smtpStatus.loading || smtpStatus.envOverride}
          aria-describedby="smtp-from-hint"
        />
        <p
          id="smtp-from-hint"
          class="text-xs text-ink-500 dark:text-ink-300 mt-1"
        >
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
          bind:value={smtpForm.tls}
          class="w-full px-4 py-2.5 border border-ink-400 dark:border-ink-400 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent focus-visible:outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          disabled={smtpStatus.loading || smtpStatus.envOverride}
        >
          <option value="starttls">STARTTLS</option>
          <option value="tls">TLS</option>
          <option value="none">None</option>
        </select>
      </div>

      {#if smtpStatus.error}
        <AlertBanner variant="error">{smtpStatus.error}</AlertBanner>
      {/if}

      {#if smtpStatus.successMessage}
        <AlertBanner variant="success">{smtpStatus.successMessage}</AlertBanner>
      {/if}

      {#if !smtpStatus.envOverride}
        <Button
          type="submit"
          disabled={smtpStatus.loading}
          class="w-full px-4 py-2.5"
        >
          {smtpSubmitLabel}
        </Button>
      {/if}
    </form>
  </div>
</div>
