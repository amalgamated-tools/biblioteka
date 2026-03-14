<script lang="ts">
  import { onMount } from "svelte";
  import { BookCheck } from "lucide-svelte";
  import { signIn, signUp } from "../stores/auth";

  let isLogin = $state(true);
  let email = $state("");
  let name = $state("");
  let password = $state("");
  let error: string | null = $state(null);
  let loading = $state(false);
  let oidcEnabled = $state(false);

  onMount(async () => {
    try {
      const res = await fetch("/api/auth/oidc/enabled");
      const data = await res.json();
      oidcEnabled = data.enabled === true;
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
      ? await signIn(email, password)
      : await signUp(name, email, password);

    if (result.error) {
      error = result.error.message;
    }

    loading = false;
  }
</script>

<div
  class="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 flex items-center justify-center p-4"
>
  <div class="w-full max-w-md">
    <div class="text-center mb-8">
      <div
        class="inline-flex items-center justify-center w-16 h-16 bg-blue-600 rounded-2xl mb-4"
      >
        <BookCheck class="w-8 h-8 text-white" />
      </div>
      <h1 class="text-3xl font-bold text-slate-900 dark:text-slate-100 mb-2">
        biblioteka
      </h1>
      <p class="text-slate-600 dark:text-slate-400">
        Manage your ebooks with ease
      </p>
    </div>

    <div
      class="bg-white dark:bg-slate-800 rounded-2xl shadow-xl dark:shadow-slate-900/30 p-8"
    >
      {#if oidcEnabled}
        <a
          href="/api/auth/oidc/login"
          class="w-full flex items-center justify-center gap-2 bg-slate-800 hover:bg-slate-900 dark:bg-slate-600 dark:hover:bg-slate-500 text-white font-medium py-3 px-4 rounded-lg transition-colors mb-6"
        >
          Login with Single Sign-On
        </a>

        <div class="relative mb-6">
          <div class="absolute inset-0 flex items-center">
            <div
              class="w-full border-t border-slate-200 dark:border-slate-700"
            ></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span
              class="px-2 bg-white dark:bg-slate-800 text-slate-500 dark:text-slate-400"
              >or</span
            >
          </div>
        </div>
      {/if}

      <div
        class="flex gap-2 mb-6 bg-slate-100 dark:bg-slate-700 rounded-lg p-1"
      >
        <button
          id="login-btn"
          onclick={() => (isLogin = true)}
          class="flex-1 py-2 px-4 rounded-md font-medium transition-all {isLogin
            ? 'bg-white dark:bg-slate-600 text-slate-900 dark:text-slate-100 shadow-sm'
            : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100'}"
        >
          Login
        </button>
        <button
          id="signup-btn"
          onclick={() => (isLogin = false)}
          class="flex-1 py-2 px-4 rounded-md font-medium transition-all {!isLogin
            ? 'bg-white dark:bg-slate-600 text-slate-900 dark:text-slate-100 shadow-sm'
            : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100'}"
        >
          Sign Up
        </button>
      </div>

      <form onsubmit={handleSubmit} class="space-y-4">
        {#if !isLogin}
          <div>
            <label
              for="name"
              class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
            >
              Name
            </label>
            <input
              id="name"
              type="text"
              bind:value={name}
              autocomplete="name"
              class="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all"
              placeholder="Your name"
              disabled={loading}
            />
          </div>
        {/if}
        <div>
          <label
            for="email"
            class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
          >
            Email
          </label>
          <input
            id="email"
            type="email"
            bind:value={email}
            autocomplete="email"
            class="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all"
            placeholder="you@example.com"
            disabled={loading}
          />
        </div>

        <div>
          <label
            for="password"
            class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
          >
            Password
          </label>
          <input
            id="password"
            type="password"
            bind:value={password}
            autocomplete={isLogin ? "current-password" : "new-password"}
            class="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all"
            placeholder="••••••••"
            disabled={loading}
          />
        </div>

        {#if error}
          <div
            class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm"
          >
            {error}
          </div>
        {/if}

        <button
          type="submit"
          disabled={loading}
          class="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-3 px-4 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? "Processing..." : isLogin ? "Sign In" : "Create Account"}
        </button>
      </form>
    </div>
  </div>
</div>
