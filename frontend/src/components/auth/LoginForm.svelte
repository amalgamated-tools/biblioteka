<script lang="ts">
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    email?: string;
    password?: string;
    emailInvalid?: boolean;
    passwordInvalid?: boolean;
    error?: string | null;
    errorVisible?: boolean;
    loading?: boolean;
    hidden?: boolean;
    onsubmit: (event: SubmitEvent) => void;
  }

  let {
    email = $bindable(""),
    password = $bindable(""),
    emailInvalid = $bindable(false),
    passwordInvalid = $bindable(false),
    error = $bindable(null) as string | null,
    errorVisible = false,
    loading = $bindable(false),
    hidden = false,
    onsubmit,
  }: Props = $props();
</script>

<div id="login-panel" role="tabpanel" aria-labelledby="login-tab" {hidden}>
  <form {onsubmit} novalidate class="space-y-4">
    <p class="text-xs text-ink-500 dark:text-ink-400">
      Fields marked with
      <span class="text-danger-600" aria-hidden="true">*</span>
      <span class="sr-only">an asterisk</span> are required.
    </p>
    <div>
      <label
        for="login-email"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Email <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="login-email"
        type="email"
        bind:value={email}
        autocomplete="email"
        required
        class="w-full py-3"
        placeholder="you@example.com"
        disabled={loading}
        aria-required={true}
        aria-invalid={emailInvalid}
        aria-describedby={emailInvalid ? "login-auth-error" : undefined}
      />
    </div>

    <div>
      <label
        for="login-password"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Password <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="login-password"
        type="password"
        bind:value={password}
        autocomplete="current-password"
        required
        class="w-full py-3"
        placeholder="••••••••"
        disabled={loading}
        aria-required={true}
        aria-invalid={passwordInvalid}
        aria-describedby={passwordInvalid ? "login-auth-error" : undefined}
      />
    </div>

    {#if errorVisible}
      <AlertBanner
        id="login-auth-error"
        variant="error"
        testId="auth-error"
        role="alert">{error}</AlertBanner
      >
    {/if}

    <Button
      type="submit"
      disabled={loading}
      class="w-full py-3 px-4 active:scale-[0.98]"
    >
      {loading ? "Processing..." : "Sign In"}
    </Button>
  </form>
</div>
