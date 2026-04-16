<script lang="ts">
  import { libraryStore } from "../../stores/libraries.svelte";
  import { routerStore } from "../../stores/router.svelte";
  import { authStore } from "../../stores/auth.svelte";
  import { onboardingStore } from "../../stores/onboarding.svelte";
  import {
    LIBRARY_ORGANIZATION_OPTIONS,
    LIBRARY_ORGANIZATION_TYPES,
    type LibraryOrganizationType,
  } from "../../types";
  import { required, validate } from "../../lib/validation";
  import {
    BookOpen,
    ChevronLeft,
    ChevronRight,
    FolderOpen,
    Plus,
    X,
  } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  const TOTAL_STEPS = 4;

  let step = $state(1);
  let saving = $state(false);
  let formError: string | null = $state(null);

  // Step 1: Name
  let formName = $state("");
  let nameError: string | null = $state(null);

  // Step 2: Paths
  let nextPathId = 0;
  let formPaths: { id: number; value: string }[] = $state([
    { id: nextPathId++, value: "" },
  ]);
  let pathsError: string | null = $state(null);

  // Step 3: Options
  let formOrganizationType = $state<LibraryOrganizationType>(
    LIBRARY_ORGANIZATION_TYPES.BOOK_PER_FOLDER,
  );
  let formMonitored = $state(false);

  const validPaths = $derived(
    formPaths.map((e) => e.value.trim()).filter((p) => p.length > 0),
  );

  const stepTitles = [
    "Name your library",
    "Choose folders",
    "Configure options",
    "Review & create",
  ];

  const stepDescriptions = [
    "Give your library a name so you can recognize it easily.",
    "Add the folder paths where your books are stored.",
    "Choose how Biblioteka should organize your files.",
    "Review your settings, then create your library.",
  ];

  function handleSkip() {
    onboardingStore.skip(authStore.user?.id);
    routerStore.navigate("dashboard");
  }

  function handleBack() {
    if (step > 1) {
      formError = null;
      step--;
    }
  }

  function handleNext() {
    formError = null;
    if (step === 1) {
      const name = formName.trim();
      nameError = validate(name, [required("Name is required")]);
      if (nameError) return;
      step = 2;
    } else if (step === 2) {
      if (validPaths.length === 0) {
        pathsError = "At least one folder is required";
        return;
      }
      pathsError = null;
      step = 3;
    } else if (step === 3) {
      step = 4;
    }
  }

  async function handleCreate() {
    saving = true;
    formError = null;
    try {
      const lib = await libraryStore.add({
        name: formName.trim(),
        paths: validPaths,
        organization_type: formOrganizationType,
        monitored: formMonitored,
      });
      routerStore.navigate(`libraries/${lib.id}`);
    } catch (e) {
      formError = e instanceof Error ? e.message : "Failed to create library";
    } finally {
      saving = false;
    }
  }
</script>

<div
  data-testid="first-library-wizard"
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6 animate-fade-in max-w-lg mx-auto"
>
  <!-- Header -->
  <div class="flex items-start justify-between mb-5">
    <div>
      <div class="flex items-center gap-2 mb-1">
        <BookOpen
          class="w-4 h-4 text-accent-600 dark:text-accent-400"
          aria-hidden="true"
        />
        <span
          class="text-xs font-semibold text-accent-600 dark:text-accent-400 uppercase tracking-wide"
        >
          First Library Setup
        </span>
      </div>
      <h1
        class="text-xl font-display font-bold text-ink-900 dark:text-cream-100"
      >
        {stepTitles[step - 1]}
      </h1>
      <p class="text-sm text-ink-500 dark:text-ink-300 mt-0.5">
        {stepDescriptions[step - 1]}
      </p>
    </div>
    <button
      onclick={handleSkip}
      class="ml-4 flex-shrink-0 text-sm text-ink-500 dark:text-ink-400 hover:text-ink-600 dark:hover:text-ink-200 transition-colors hover:underline underline-offset-2"
      aria-label="Skip first library setup for now"
      disabled={saving}
    >
      Skip for now
    </button>
  </div>

  <!-- Step progress bar -->
  <div
    class="flex items-center gap-1.5 mb-6"
    role="group"
    aria-label="Progress: step {step} of {TOTAL_STEPS}"
  >
    {#each Array.from({ length: TOTAL_STEPS }, (_, i) => i + 1) as s (s)}
      <div
        class="h-1.5 flex-1 rounded-full transition-all {s <= step
          ? 'bg-accent-600 dark:bg-accent-500'
          : 'bg-ink-100 dark:bg-ink-700'}"
        aria-hidden="true"
      ></div>
    {/each}
  </div>

  <!-- Live region for assistive technology -->
  <p role="status" class="sr-only">
    Step {step} of {TOTAL_STEPS}: {stepTitles[step - 1]}
  </p>

  {#if formError}
    <AlertBanner variant="error" class="mb-4">{formError}</AlertBanner>
  {/if}

  <!-- Step 1: Name -->
  {#if step === 1}
    <div>
      <label
        for="wizard-lib-name"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
      >
        Library Name
        <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="wizard-lib-name"
        bind:value={formName}
        placeholder="e.g. Fiction, Non-Fiction, Audiobooks"
        class="w-full py-2.5"
        disabled={saving}
        aria-required={true}
        aria-invalid={nameError ? true : undefined}
        aria-describedby={nameError ? "wizard-name-error" : undefined}
      />
      {#if nameError}
        <p
          id="wizard-name-error"
          role="alert"
          class="text-sm text-danger-600 dark:text-red-400 mt-1"
        >
          {nameError}
        </p>
      {/if}
    </div>
  {/if}

  <!-- Step 2: Paths -->
  {#if step === 2}
    <fieldset class="border-none p-0 m-0">
      <legend
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
      >
        Folders
        <span class="text-danger-600" aria-hidden="true">*</span>
      </legend>
      <div class="space-y-2">
        {#each formPaths as entry, i (entry.id)}
          <div class="flex items-center gap-2">
            <FolderOpen
              class="w-4 h-4 text-ink-300 flex-shrink-0"
              aria-hidden="true"
            />
            <TextInput
              id={`wizard-folder-${i}`}
              value={entry.value}
              oninput={(e) => {
                formPaths[i] = {
                  ...formPaths[i],
                  value: e.currentTarget.value,
                };
              }}
              aria-label={formPaths.length === 1
                ? "Folder path"
                : `Folder path ${i + 1}`}
              placeholder="/path/to/books"
              class="flex-1 py-2.5 font-mono text-sm"
              disabled={saving}
              aria-invalid={pathsError ? true : undefined}
              aria-describedby={pathsError ? "wizard-paths-error" : undefined}
            />
            {#if formPaths.length > 1}
              <button
                type="button"
                onclick={() => {
                  formPaths = formPaths.filter((_, idx) => idx !== i);
                }}
                class="p-2 text-ink-400 hover:text-danger-600 transition-colors"
                aria-label={entry.value
                  ? `Remove folder "${entry.value}"`
                  : `Remove folder ${i + 1}`}
                disabled={saving}
              >
                <X class="w-4 h-4" aria-hidden="true" />
              </button>
            {/if}
          </div>
        {/each}
      </div>
      <button
        type="button"
        onclick={() => {
          formPaths = [...formPaths, { id: nextPathId++, value: "" }];
        }}
        class="mt-2 inline-flex items-center gap-1.5 text-sm text-accent-600 dark:text-accent-400 hover:text-accent-700 dark:hover:text-accent-300 transition-colors font-medium"
        disabled={saving}
      >
        <Plus class="w-3.5 h-3.5" aria-hidden="true" />
        Add another folder
      </button>
      {#if pathsError}
        <p
          id="wizard-paths-error"
          role="alert"
          class="text-sm text-danger-600 dark:text-red-400 mt-1"
        >
          {pathsError}
        </p>
      {/if}
    </fieldset>
  {/if}

  <!-- Step 3: Options -->
  {#if step === 3}
    <div class="space-y-5">
      <div>
        <label
          for="wizard-org-type"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          File Organization
        </label>
        <select
          id="wizard-org-type"
          bind:value={formOrganizationType}
          class="w-full px-4 py-2.5 border border-ink-400 dark:border-ink-400 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent focus-visible:outline-none dark:bg-ink-800 dark:text-cream-100 transition-all"
          disabled={saving}
        >
          {#each LIBRARY_ORGANIZATION_OPTIONS as option (option.value)}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>
        <p class="text-xs text-ink-500 dark:text-ink-400 mt-1">
          Determines how Biblioteka organizes books it imports into this
          library.
        </p>
      </div>
      <div class="flex items-center">
        <label
          for="wizard-monitored"
          class="relative inline-flex items-center gap-3 cursor-pointer"
        >
          <input
            id="wizard-monitored"
            type="checkbox"
            role="switch"
            aria-checked={formMonitored}
            bind:checked={formMonitored}
            class="sr-only peer"
            disabled={saving}
          />
          <div
            class="relative w-11 h-6 bg-ink-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-accent-500 dark:bg-ink-700 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-ink-200 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-ink-600 peer-checked:bg-accent-600"
          ></div>
          <span class="text-sm text-ink-600 dark:text-ink-300"
            >Monitor for new content</span
          >
        </label>
      </div>
    </div>
  {/if}

  <!-- Step 4: Review -->
  {#if step === 4}
    <dl class="space-y-4">
      <div class="flex gap-3">
        <dt
          class="text-sm font-medium text-ink-500 dark:text-ink-400 w-32 flex-shrink-0"
        >
          Name
        </dt>
        <dd class="text-sm text-ink-900 dark:text-cream-100 font-medium">
          {formName.trim() || "—"}
        </dd>
      </div>
      <div class="flex gap-3">
        <dt
          class="text-sm font-medium text-ink-500 dark:text-ink-400 w-32 flex-shrink-0"
        >
          Folders
        </dt>
        <dd class="text-sm text-ink-900 dark:text-cream-100">
          {#if validPaths.length > 0}
            <ul class="space-y-0.5 font-mono">
              {#each validPaths as path (path)}
                <li>{path}</li>
              {/each}
            </ul>
          {:else}
            <span class="text-ink-500 dark:text-ink-400">—</span>
          {/if}
        </dd>
      </div>
      <div class="flex gap-3">
        <dt
          class="text-sm font-medium text-ink-500 dark:text-ink-400 w-32 flex-shrink-0"
        >
          Organization
        </dt>
        <dd class="text-sm text-ink-900 dark:text-cream-100">
          {LIBRARY_ORGANIZATION_OPTIONS.find(
            (o) => o.value === formOrganizationType,
          )?.label ?? formOrganizationType}
        </dd>
      </div>
      <div class="flex gap-3">
        <dt
          class="text-sm font-medium text-ink-500 dark:text-ink-400 w-32 flex-shrink-0"
        >
          Monitoring
        </dt>
        <dd class="text-sm text-ink-900 dark:text-cream-100">
          {formMonitored ? "Enabled" : "Disabled"}
        </dd>
      </div>
    </dl>
  {/if}

  <!-- Navigation -->
  <div
    class="flex items-center justify-between mt-6 pt-5 border-t border-ink-100 dark:border-ink-800"
  >
    <div>
      {#if step > 1}
        <Button
          variant="secondary"
          onclick={handleBack}
          disabled={saving}
          class="inline-flex items-center px-4 py-2.5 text-sm"
        >
          <ChevronLeft class="w-4 h-4 mr-1" aria-hidden="true" />
          Back
        </Button>
      {/if}
    </div>
    <div>
      {#if step < TOTAL_STEPS}
        <Button
          onclick={handleNext}
          disabled={saving}
          class="inline-flex items-center px-5 py-2.5 text-sm active:scale-[0.98]"
        >
          Next
          <ChevronRight class="w-4 h-4 ml-1" aria-hidden="true" />
        </Button>
      {:else}
        <Button
          onclick={handleCreate}
          disabled={saving}
          class="px-5 py-2.5 text-sm active:scale-[0.98]"
        >
          {saving ? "Creating…" : "Create Library"}
        </Button>
      {/if}
    </div>
  </div>
</div>
