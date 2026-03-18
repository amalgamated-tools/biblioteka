<script lang="ts">
  import { onMount } from "svelte";
  import { BookCheck } from "lucide-svelte";
  import { authStore } from "../stores/auth.svelte";
  import { getOidcEnabled } from "../lib/api";
  import AlertBanner from "./ui/AlertBanner.svelte";

  let isLogin = $state(true);
  let email = $state("");
  let name = $state("");
  let password = $state("");
  let error: string | null = $state(null);
  let loading = $state(false);
  let oidcEnabled = $state(false);

  function handleTabKeydown(event: KeyboardEvent) {
    if (loading) return;
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault();
      isLogin = !isLogin;
      const nextId = isLogin ? 'login-tab' : 'signup-tab';
      document.getElementById(nextId)?.focus();
    } else if (event.key === 'Home') {
      event.preventDefault();
      isLogin = true;
      document.getElementById('login-tab')?.focus();
    } else if (event.key === 'End') {
      event.preventDefault();
      isLogin = false;
      document.getElementById('signup-tab')?.focus();
    }
  }

  onMount(async () => {
    try {
      oidcEnabled = await getOidcEnabled();
    } catch {
      // OIDC not available
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    loading = true;

    if (!email || !password || (!isLogin && !name)) {
      error = "Please fill in all fields";
      loading = false;
      return;
    }

    if (password.length < 6) {
      error = "Password must be at least 6 characters";
      loading = false;
      return;
    }

    const result = isLogin
      ? await authStore.signIn(email, password)
      : await authStore.signUp(name, email, password);

    if (result.error) {
      error = result.error.message;
    }

    loading = false;
  }
</script>

<a
  href="#auth-main"
  onclick={(e: MouseEvent) => {
    e.preventDefault();
    document.getElementById("auth-main")?.focus();
  }}
  class="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-xl focus:bg-accent-600 focus:px-4 focus:py-2 focus:font-semibold focus:text-white"
>
  Skip to main content
</a>

<main
  id="auth-main"
  tabindex="-1"
  class="min-h-screen bg-cream-50 dark:bg-ink-950 flex items-center justify-center p-4 relative bg-texture"
>
  <!-- Decorative background elements -->
  <div class="absolute inset-0 overflow-hidden pointer-events-none">
    <div
      class="absolute -top-24 -right-24 w-96 h-96 rounded-full bg-accent-200/30 dark:bg-accent-800/10 blur-3xl"
    ></div>
    <div
      class="absolute -bottom-32 -left-32 w-[500px] h-[500px] rounded-full bg-accent-100/40 dark:bg-accent-800/5 blur-3xl"
    ></div>
  </div>

  <div class="w-full max-w-md relative z-10 animate-fade-in-up">
    <div class="text-center mb-8">
      <div
        class="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-accent-500 to-accent-700 rounded-2xl mb-4 shadow-lg shadow-accent-500/20 dark:shadow-accent-500/10"
      >
        <BookCheck class="w-8 h-8 text-white" />
      </div>
      <h1
        class="text-4xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2"
      >
        biblioteka
      </h1>
      <p class="text-ink-400 dark:text-ink-400 font-body">
        Your personal digital library
      </p>
    </div>

    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-xl shadow-ink-900/5 dark:shadow-ink-950/50 border border-ink-100 dark:border-ink-800 p-8"
    >
      {#if oidcEnabled}
        <a
          href="/api/auth/oidc/login"
          class="w-full flex items-center justify-center gap-2 bg-ink-800 hover:bg-ink-900 dark:bg-ink-700 dark:hover:bg-ink-600 text-white font-medium py-3 px-4 rounded-xl transition-all hover:shadow-lg mb-6"
        >
          Login with Single Sign-On
        </a>

        <div class="relative mb-6">
          <div class="absolute inset-0 flex items-center">
            <div
              class="w-full border-t border-ink-100 dark:border-ink-700"
            ></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span
              class="px-3 bg-white dark:bg-ink-900 text-ink-400 dark:text-ink-400"
              >or</span
            >
          </div>
        </div>
      {/if}

      <!-- svelte-ignore a11y_interactive_supports_focus -->
      <div role="tablist" aria-label="Authentication method" onkeydown={handleTabKeydown} class="flex gap-1 mb-6 bg-cream-100 dark:bg-ink-800 rounded-xl p-1">
        <button
          id="login-tab"
          role="tab"
          aria-selected={isLogin}
          aria-controls="login-panel"
          tabindex={isLogin ? 0 : -1}
          disabled={loading}
          onclick={() => (isLogin = true)}
          class="flex-1 py-2.5 px-4 rounded-lg font-medium transition-all {isLogin
            ? 'bg-white dark:bg-ink-700 text-ink-900 dark:text-cream-100 shadow-sm'
            : 'text-ink-400 dark:text-ink-400 hover:text-ink-700 dark:hover:text-ink-200'}"
        >
          Login
        </button>
        <button
          id="signup-tab"
          role="tab"
          aria-selected={!isLogin}
          aria-controls="signup-panel"
          tabindex={!isLogin ? 0 : -1}
          disabled={loading}
          onclick={() => (isLogin = false)}
          class="flex-1 py-2.5 px-4 rounded-lg font-medium transition-all {!isLogin
            ? 'bg-white dark:bg-ink-700 text-ink-900 dark:text-cream-100 shadow-sm'
            : 'text-ink-400 dark:text-ink-400 hover:text-ink-700 dark:hover:text-ink-200'}"
        >
          Sign Up
        </button>
      </div>

      <div id="login-panel" role="tabpanel" tabindex="0" aria-labelledby="login-tab" hidden={!isLogin}>
        <form onsubmit={handleSubmit} class="space-y-4">
          <div>
            <label
              for="login-email"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Email
            </label>
            <input
              id="login-email"
              type="email"
              bind:value={email}
              autocomplete="email"
              class="w-full px-4 py-3 rounded-xl border border-ink-200 dark:border-ink-700 bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
              placeholder="you@example.com"
              disabled={loading}
            />
          </div>

          <div>
            <label
              for="login-password"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Password
            </label>
            <input
              id="login-password"
              type="password"
              bind:value={password}
              autocomplete="current-password"
              class="w-full px-4 py-3 rounded-xl border border-ink-200 dark:border-ink-700 bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
              placeholder="••••••••"
              disabled={loading}
            />
          </div>

          {#if error && isLogin}
            <AlertBanner variant="error" testId="auth-error" role="alert"
              >{error}</AlertBanner
            >
          {/if}

          <button
            type="submit"
            disabled={loading}
            class="w-full bg-gradient-to-r from-accent-600 to-accent-700 hover:from-accent-700 hover:to-accent-800 text-white font-semibold py-3 px-4 rounded-xl transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-md shadow-accent-600/20 hover:shadow-lg hover:shadow-accent-600/30 active:scale-[0.98]"
          >
            {loading ? "Processing..." : "Sign In"}
          </button>
        </form>
      </div>

      <div id="signup-panel" role="tabpanel" tabindex="0" aria-labelledby="signup-tab" hidden={isLogin}>
        <form onsubmit={handleSubmit} class="space-y-4">
          <div>
            <label
              for="signup-name"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Name
            </label>
            <input
              id="signup-name"
              type="text"
              bind:value={name}
              autocomplete="name"
              class="w-full px-4 py-3 rounded-xl border border-ink-200 dark:border-ink-700 bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
              placeholder="Your name"
              disabled={loading}
            />
          </div>

          <div>
            <label
              for="signup-email"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Email
            </label>
            <input
              id="signup-email"
              type="email"
              bind:value={email}
              autocomplete="email"
              class="w-full px-4 py-3 rounded-xl border border-ink-200 dark:border-ink-700 bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
              placeholder="you@example.com"
              disabled={loading}
            />
          </div>

          <div>
            <label
              for="signup-password"
              class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
            >
              Password
            </label>
            <input
              id="signup-password"
              type="password"
              bind:value={password}
              autocomplete="new-password"
              class="w-full px-4 py-3 rounded-xl border border-ink-200 dark:border-ink-700 bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500"
              placeholder="••••••••"
              disabled={loading}
            />
          </div>

          {#if error && !isLogin}
            <AlertBanner variant="error" testId="auth-error" role="alert"
              >{error}</AlertBanner
            >
          {/if}

          <button
            type="submit"
            disabled={loading}
            class="w-full bg-gradient-to-r from-accent-600 to-accent-700 hover:from-accent-700 hover:to-accent-800 text-white font-semibold py-3 px-4 rounded-xl transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-md shadow-accent-600/20 hover:shadow-lg hover:shadow-accent-600/30 active:scale-[0.98]"
          >
            {loading ? "Processing..." : "Create Account"}
          </button>
        </form>
      </div>
    </div>
  </div>
</main>
