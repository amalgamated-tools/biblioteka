<script lang="ts">
  import { authStore } from "../../stores/auth.svelte";
  import { Mail } from "lucide-svelte";
  import TextInput from "../ui/TextInput.svelte";
</script>

<div>
  <h2
    class="text-xl font-display font-bold text-ink-900 dark:text-cream-100 mb-4 flex items-center gap-2"
  >
    <Mail class="w-5 h-5 text-accent-600" aria-hidden="true" />
    Account Information
  </h2>
  <div class="space-y-4">
    <div>
      <label
        for="email-display"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
      >
        Email Address
      </label>
      <TextInput
        id="email-display"
        type="email"
        value={authStore.user?.email || ""}
        disabled
        class="w-full py-2.5"
        aria-describedby="email-display-hint"
      />
      <p
        id="email-display-hint"
        class="text-xs text-ink-500 dark:text-ink-300 mt-1"
      >
        Contact support to change your email address
      </p>
    </div>
  </div>
</div>
