<script lang="ts">
  import { changePassword } from "../../lib/api";
  import { required, minLength, matches, validate } from "../../lib/validation";
  import { AutoDismissTimer } from "../../lib/autoDismissTimer.svelte";
  import { Lock } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let passwordError: string | null = $state(null);
  let passwordLoading = $state(false);
  const successTimer = new AutoDismissTimer();
  let currentPasswordInvalid = $derived(
    passwordError === "Current password is required",
  );
  let newPasswordInvalid = $derived(
    passwordError === "New password is required" ||
      passwordError === "New password must be at least 6 characters",
  );
  let confirmPasswordInvalid = $derived(
    passwordError === "Passwords do not match",
  );

  $effect(() => {
    return () => {
      successTimer.clear();
    };
  });

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
        aria-invalid={currentPasswordInvalid}
        aria-describedby={currentPasswordInvalid
          ? "password-change-error"
          : undefined}
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
        aria-invalid={newPasswordInvalid}
        aria-describedby={newPasswordInvalid
          ? "password-change-error"
          : undefined}
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
        aria-invalid={confirmPasswordInvalid}
        aria-describedby={confirmPasswordInvalid
          ? "password-change-error"
          : undefined}
      />
    </div>

    {#if passwordError}
      <AlertBanner id="password-change-error" variant="error"
        >{passwordError}</AlertBanner
      >
    {/if}

    {#if successTimer.visible}
      <AlertBanner variant="success">Password updated successfully</AlertBanner>
    {/if}

    <Button type="submit" disabled={passwordLoading} class="w-full px-4 py-2.5">
      {passwordLoading ? "Updating..." : "Update Password"}
    </Button>
  </form>
</div>
