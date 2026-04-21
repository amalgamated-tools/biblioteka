<script lang="ts">
  import { BookCheck } from "lucide-svelte";
  import { authStore } from "../stores/auth.svelte";
  import {
    beginPasskeyLogin,
    finishPasskeyLogin,
    prepareRequestOptions,
  } from "../lib/api";
  import { fetchAuthFeatureFlags } from "../lib/authFeatureFlags";
  import {
    validateAuthForm,
    type AuthFormValidationResult,
  } from "../lib/authFormValidation";
  import {
    getLoginFieldInvalidState,
    getSignupFieldInvalidState,
  } from "../lib/authErrors";
  import AlertBanner from "./ui/AlertBanner.svelte";
  import LoginForm from "./auth/LoginForm.svelte";
  import SignupForm from "./auth/SignupForm.svelte";

  let isLogin = $state(true);
  let email = $state("");
  let name = $state("");
  let password = $state("");
  let error: string | null = $state(null);
  let loading = $state(false);
  let oidcEnabled = $state(false);
  let signupEnabled = $state(true);
  let passkeyEnabled = $state(false);
  let initError: string | null = $state(null);
  let passkeyLoading = $state(false);
  let loginEmailInvalid = $state(false);
  let loginPasswordInvalid = $state(false);
  let signupNameInvalid = $state(false);
  let signupEmailInvalid = $state(false);
  let signupPasswordInvalid = $state(false);
  let loginErrorVisible = $derived(!!error && isLogin);
  let signupErrorVisible = $derived(!!error && !isLogin);

  function handleTabKeydown(event: KeyboardEvent) {
    if (loading || !signupEnabled) return;
    if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
      event.preventDefault();
      isLogin = !isLogin;
      document.getElementById(isLogin ? "login-tab" : "signup-tab")?.focus();
    } else if (event.key === "Home") {
      event.preventDefault();
      isLogin = true;
      document.getElementById("login-tab")?.focus();
    } else if (event.key === "End") {
      event.preventDefault();
      isLogin = false;
      document.getElementById("signup-tab")?.focus();
    }
  }

  $effect(() => {
    const controller = new AbortController();

    async function initAuth() {
      const flags = await fetchAuthFeatureFlags(controller.signal);
      if (controller.signal.aborted) return;
      oidcEnabled = flags.oidcEnabled;
      signupEnabled = flags.signupEnabled;
      passkeyEnabled = flags.passkeyEnabled;
      initError = flags.initError;
      if (!flags.signupEnabled) isLogin = true;
    }

    initAuth();
    return () => controller.abort();
  });

  async function handlePasskeySignIn() {
    error = null;
    passkeyLoading = true;

    try {
      const { session_id, options } = await beginPasskeyLogin();
      const assertion = await navigator.credentials.get({
        publicKey: prepareRequestOptions(options),
      });

      if (!assertion || !(assertion instanceof PublicKeyCredential)) {
        error = "No passkey was selected";
        return;
      }

      const assertionJSON = (
        assertion as PublicKeyCredential & { toJSON(): unknown }
      ).toJSON();
      const result = await finishPasskeyLogin(session_id, assertionJSON);
      authStore.user = result.user;
    } catch (err) {
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        error = null;
      } else {
        error = err instanceof Error ? err.message : "Passkey sign-in failed";
      }
    } finally {
      passkeyLoading = false;
    }
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    error = null;
    loading = true;
    applyValidationState(validateAuthForm({ isLogin, name, email, password }));
    if (error) {
      loading = false;
      return;
    }

    const result = isLogin
      ? await authStore.signIn(email, password)
      : await authStore.signUp(name, email, password);

    if (result.error) {
      error = result.error.message;
      if (isLogin) {
        const invalidState = getLoginFieldInvalidState(result.error.message);
        loginEmailInvalid = invalidState.email;
        loginPasswordInvalid = invalidState.password;
      } else {
        const invalidState = getSignupFieldInvalidState(result.error.message);
        signupNameInvalid = invalidState.name;
        signupEmailInvalid = invalidState.email;
        signupPasswordInvalid = invalidState.password;
      }
    }

    loading = false;
  }

  function applyValidationState(state: AuthFormValidationResult) {
    error = state.error;
    loginEmailInvalid = state.loginEmailInvalid;
    loginPasswordInvalid = state.loginPasswordInvalid;
    signupNameInvalid = state.signupNameInvalid;
    signupEmailInvalid = state.signupEmailInvalid;
    signupPasswordInvalid = state.signupPasswordInvalid;
  }
</script>

<div
  class="min-h-screen bg-cream-50 dark:bg-ink-950 flex items-center justify-center p-4 relative bg-texture"
>
  <div class="absolute inset-0 overflow-hidden pointer-events-none">
    <div
      class="absolute -top-24 -right-24 w-96 h-96 rounded-full bg-accent-200/30 dark:bg-accent-800/10 blur-3xl"
    ></div>
    <div
      class="absolute -bottom-32 -left-32 w-[500px] h-[500px] rounded-full bg-accent-100/40 dark:bg-accent-800/5 blur-3xl"
    ></div>
  </div>

  <main class="w-full max-w-md relative z-10 animate-fade-in-up">
    <div class="text-center mb-8">
      <div
        class="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-accent-500 to-accent-700 rounded-2xl mb-4 shadow-lg shadow-accent-500/20 dark:shadow-accent-500/10"
      >
        <BookCheck class="w-8 h-8 text-white" aria-hidden="true" />
      </div>
      <h1
        class="text-4xl font-display font-bold text-ink-900 dark:text-cream-100 mb-2"
      >
        biblioteka
      </h1>
      <p class="text-ink-500 dark:text-ink-300 font-body">
        Your personal digital library
      </p>
    </div>

    <div
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-xl shadow-ink-900/5 dark:shadow-ink-950/50 border border-ink-100 dark:border-ink-800 p-8"
    >
      {#if initError}
        <AlertBanner variant="error" class="mb-6">{initError}</AlertBanner>
      {/if}

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
              class="px-3 bg-white dark:bg-ink-900 text-ink-500 dark:text-ink-300"
              >or</span
            >
          </div>
        </div>
      {/if}

      {#if passkeyEnabled}
        <button
          type="button"
          disabled={loading || passkeyLoading}
          onclick={handlePasskeySignIn}
          class="w-full flex items-center justify-center gap-2 bg-accent-600 hover:bg-accent-700 dark:bg-accent-700 dark:hover:bg-accent-600 text-white font-medium py-3 px-4 rounded-xl transition-all hover:shadow-lg mb-6 disabled:opacity-60 disabled:cursor-not-allowed"
        >
          {passkeyLoading ? "Waiting for passkey..." : "Sign in with a Passkey"}
        </button>

        <div class="relative mb-6">
          <div class="absolute inset-0 flex items-center">
            <div
              class="w-full border-t border-ink-100 dark:border-ink-700"
            ></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span
              class="px-3 bg-white dark:bg-ink-900 text-ink-500 dark:text-ink-300"
              >or sign in with password</span
            >
          </div>
        </div>
      {/if}

      <!-- svelte-ignore a11y_interactive_supports_focus -->
      <div
        role="tablist"
        aria-label="Authentication method"
        onkeydown={handleTabKeydown}
        class="flex gap-1 mb-6 bg-cream-100 dark:bg-ink-800 rounded-xl p-1"
      >
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
            : 'text-ink-500 dark:text-ink-300 hover:text-ink-700 dark:hover:text-ink-200'}"
        >
          Login
        </button>
        {#if signupEnabled}
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
              : 'text-ink-500 dark:text-ink-300 hover:text-ink-700 dark:hover:text-ink-200'}"
          >
            Sign Up
          </button>
        {/if}
      </div>

      <LoginForm
        bind:email
        bind:password
        bind:emailInvalid={loginEmailInvalid}
        bind:passwordInvalid={loginPasswordInvalid}
        bind:error
        errorVisible={loginErrorVisible}
        bind:loading
        hidden={!isLogin}
        onsubmit={handleSubmit}
      />

      {#if signupEnabled}
        <SignupForm
          bind:name
          bind:email
          bind:password
          bind:nameInvalid={signupNameInvalid}
          bind:emailInvalid={signupEmailInvalid}
          bind:passwordInvalid={signupPasswordInvalid}
          bind:error
          errorVisible={signupErrorVisible}
          bind:loading
          hidden={isLogin}
          onsubmit={handleSubmit}
        />
      {/if}
    </div>
  </main>
</div>
