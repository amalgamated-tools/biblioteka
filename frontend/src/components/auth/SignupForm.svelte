<script lang="ts">
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
    name?: string;
    email?: string;
    password?: string;
    nameInvalid?: boolean;
    emailInvalid?: boolean;
    passwordInvalid?: boolean;
    error?: string | null;
    errorVisible?: boolean;
    loading?: boolean;
    hidden?: boolean;
    onsubmit: (event: SubmitEvent) => void;
  }

  let {
    name = $bindable(""),
    email = $bindable(""),
    password = $bindable(""),
    nameInvalid = $bindable(false),
    emailInvalid = $bindable(false),
    passwordInvalid = $bindable(false),
    error = $bindable(null) as string | null,
    errorVisible = $bindable(false),
    loading = $bindable(false),
    hidden = false,
    onsubmit,
  }: Props = $props();
</script>

<div id="signup-panel" role="tabpanel" aria-labelledby="signup-tab" {hidden}>
  <form {onsubmit} novalidate class="space-y-4">
    <p class="text-xs text-ink-500 dark:text-ink-400">
      Fields marked with
      <span class="text-danger-600" aria-hidden="true">*</span>
      <span class="sr-only">an asterisk</span> are required.
    </p>
    <div>
      <label
        for="signup-name"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Name <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="signup-name"
        type="text"
        bind:value={name}
        autocomplete="name"
        required
        class="w-full py-3"
        placeholder="Your name"
        disabled={loading}
        aria-required={true}
        aria-invalid={nameInvalid}
        aria-describedby={nameInvalid ? "signup-auth-error" : undefined}
      />
    </div>

    <div>
      <label
        for="signup-email"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Email <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="signup-email"
        type="email"
        bind:value={email}
        autocomplete="email"
        required
        class="w-full py-3"
        placeholder="you@example.com"
        disabled={loading}
        aria-required={true}
        aria-invalid={emailInvalid}
        aria-describedby={emailInvalid ? "signup-auth-error" : undefined}
      />
    </div>

    <div>
      <label
        for="signup-password"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Password <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="signup-password"
        type="password"
        bind:value={password}
        autocomplete="new-password"
        required
        class="w-full py-3"
        placeholder="••••••••"
        disabled={loading}
        aria-required={true}
        aria-invalid={passwordInvalid}
        aria-describedby={passwordInvalid ? "signup-auth-error" : undefined}
      />
    </div>

    {#if errorVisible}
      <AlertBanner
        id="signup-auth-error"
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
      {loading ? "Processing..." : "Create Account"}
    </Button>
  </form>
</div>
