<script lang="ts">
  import { authStore } from "../../stores/auth.svelte";
  import { updateProfile } from "../../lib/api";
  import { required, validate } from "../../lib/validation";
  import { AutoDismissTimer } from "../../lib/autoDismissTimer.svelte";
  import { User } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  let displayName = $state(authStore.user?.name ?? "");
  let nameError: string | null = $state(null);
  let nameLoading = $state(false);
  const nameSuccessTimer = new AutoDismissTimer();
  let displayNameInvalid = $derived(!!nameError);

  $effect(() => {
    if (authStore.user && !nameLoading) {
      displayName = authStore.user.name;
    }
  });

  $effect(() => {
    return () => {
      nameSuccessTimer.clear();
    };
  });

  async function handleNameUpdate(e: SubmitEvent) {
    e.preventDefault();
    nameError = null;
    nameSuccessTimer.clear();

    nameError = validate(displayName, [required("Display name is required")]);
    if (nameError) return;

    nameLoading = true;
    try {
      const updated = await updateProfile(displayName);
      if (authStore.user) {
        authStore.user = { ...authStore.user, name: updated.name };
      }
      nameSuccessTimer.show();
    } catch (err) {
      nameError =
        err instanceof Error ? err.message : "Failed to update display name";
    } finally {
      nameLoading = false;
    }
  }
</script>

<div>
  <h2
    class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
  >
    <User class="w-5 h-5 text-accent-600" aria-hidden="true" />
    Display Name
  </h2>
  <form
    onsubmit={handleNameUpdate}
    aria-label="Update display name"
    class="space-y-4"
  >
    <div>
      <label
        for="display-name"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Name
      </label>
      <TextInput
        id="display-name"
        type="text"
        bind:value={displayName}
        autocomplete="name"
        class="w-full py-2.5"
        placeholder="Your name"
        disabled={nameLoading}
        aria-invalid={displayNameInvalid}
        aria-describedby={displayNameInvalid ? "display-name-error" : undefined}
      />
    </div>

    {#if nameError}
      <AlertBanner id="display-name-error" variant="error"
        >{nameError}</AlertBanner
      >
    {/if}

    {#if nameSuccessTimer.visible}
      <AlertBanner variant="success">Display name updated</AlertBanner>
    {/if}

    <Button type="submit" disabled={nameLoading} class="w-full">
      {nameLoading ? "Saving..." : "Save Name"}
    </Button>
  </form>
</div>
