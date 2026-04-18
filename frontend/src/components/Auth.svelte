<script lang="ts">
  import { BookCheck } from "lucide-svelte";
  import { authStore } from "../stores/auth.svelte";
  import {
    getOidcEnabled,
    getSignupEnabled,
    getPasskeyEnabled,
    beginPasskeyLogin,
    finishPasskeyLogin,
    prepareRequestOptions,
  } from "../lib/api";
  import {
    required,
    minLength,
    email as emailRule,
    validate,
  } from "../lib/validation";
  import AlertBanner from "./ui/AlertBanner.svelte";
  import Button from "./ui/Button.svelte";
  import TextInput from "./ui/TextInput.svelte";

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
  let loginErrorVisible = $derived(!!error && isLogin);
  let signupErrorVisible = $derived(!!error && !isLogin);

  function getLoginFieldInvalidState(message: string): {
    email: boolean;
    password: boolean;
  } {
    const loweredError = message.toLowerCase();

    // Deliberately ambiguous messages (e.g. anti-enumeration) — don't mark any field.
    const ambiguous = [
      /\bemail\s+or\s+password\b/,
      /\bemail\s+and\s+password\b/,
    ].some((pattern) => pattern.test(loweredError));
    if (ambiguous) {
      return { email: false, password: false };
    }

    const mentionsEmail = [
      /\binvalid email\b/,
      /\bemail is invalid\b/,
      /\bemail is not valid\b/,
      /\bunknown account\b/,
      /\baccount not found\b/,
      /\buser not found\b/,
    ].some((pattern) => pattern.test(loweredError));
    const mentionsPassword = [
      /\bpassword must\b/,
      /\binvalid password\b/,
      /\bincorrect password\b/,
      /\bwrong password\b/,
    ].some((pattern) => pattern.test(loweredError));
    const mentionsCredentials = [
      /\binvalid credentials\b/,
      /\bincorrect credentials\b/,
      /\bwrong credentials\b/,
    ].some((pattern) => pattern.test(loweredError));

    if (mentionsEmail && mentionsPassword) {
      return { email: true, password: true };
    }
    if (mentionsEmail) {
      return { email: true, password: false };
    }
    if (mentionsPassword) {
      return { email: false, password: true };
    }
    if (mentionsCredentials) {
      return { email: true, password: true };
    }

    return { email: false, password: false };
  }

  function handleTabKeydown(event: KeyboardEvent) {
    if (loading) return;
    if (!signupEnabled) return;
    if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
      event.preventDefault();
      isLogin = !isLogin;
      const nextId = isLogin ? "login-tab" : "signup-tab";
      document.getElementById(nextId)?.focus();
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

  let authInitialized = false;
  $effect(() => {
    if (!authInitialized) {
      authInitialized = true;
      getOidcEnabled()
        .then((enabled) => {
          oidcEnabled = enabled;
        })
        .catch(() => {
          if (import.meta.env.DEV) {
            console.error("Failed to check OIDC status");
          }
          initError ??= "Unable to reach the server to load auth settings";
        });
      getSignupEnabled()
        .then((enabled) => {
          signupEnabled = enabled;
          if (!enabled) {
            isLogin = true;
          }
        })
        .catch(() => {
          if (import.meta.env.DEV) {
            console.error("Failed to check signup enabled status");
          }
          initError ??= "Unable to reach the server to load auth settings";
        });
      getPasskeyEnabled()
        .then((enabled) => {
          passkeyEnabled = enabled;
        })
        .catch(() => {
          // Passkey availability is optional; silently disable the feature on error.
          passkeyEnabled = false;
        });
    }
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

      // Serialize the credential to JSON for the server.
      // PublicKeyCredential.toJSON() is available in all modern browsers.
      const assertionJSON = (
        assertion as PublicKeyCredential & { toJSON(): unknown }
      ).toJSON();

      const result = await finishPasskeyLogin(session_id, assertionJSON);
      authStore.user = result.user;
    } catch (err) {
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        // User cancelled or no passkey available — don't surface as an error.
        error = null;
      } else {
        error = err instanceof Error ? err.message : "Passkey sign-in failed";
      }
    } finally {
      passkeyLoading = false;
    }
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    loading = true;
    loginEmailInvalid = false;
    loginPasswordInvalid = false;

    if (isLogin) {
      loginEmailInvalid = required()(email) !== null;
      loginPasswordInvalid = required()(password) !== null;
      if (loginEmailInvalid || loginPasswordInvalid) {
        if (loginEmailInvalid && loginPasswordInvalid) {
          error = "Please fill in all fields";
        } else if (loginEmailInvalid) {
          error = "Please fill in the email field";
        } else {
          error = "Please fill in the password field";
        }
        loading = false;
        return;
      }
    } else if ([name, email, password].some((f) => required()(f) !== null)) {
      error = "Please fill in all fields";
      loading = false;
      return;
    }

    const pwdError = validate(password, [
      minLength(6, "Password must be at least 6 characters"),
    ]);
    if (pwdError) {
      error = pwdError;
      if (isLogin) {
        loginPasswordInvalid = true;
      }
      loading = false;
      return;
    }

    const emailError = validate(email, [
      emailRule("Please enter a valid email address"),
    ]);
    if (emailError) {
      error = emailError;
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
      }
    }

    loading = false;
  }
</script>

<div
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

      <div
        id="login-panel"
        role="tabpanel"
        aria-labelledby="login-tab"
        hidden={!isLogin}
      >
        <form onsubmit={handleSubmit} novalidate class="space-y-4">
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
              aria-invalid={loginEmailInvalid}
              aria-describedby={loginEmailInvalid
                ? "login-auth-error"
                : undefined}
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
              aria-invalid={loginPasswordInvalid}
              aria-describedby={loginPasswordInvalid
                ? "login-auth-error"
                : undefined}
            />
          </div>

          {#if loginErrorVisible}
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

      {#if signupEnabled}
        <div
          id="signup-panel"
          role="tabpanel"
          aria-labelledby="signup-tab"
          hidden={isLogin}
        >
          <form onsubmit={handleSubmit} novalidate class="space-y-4">
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
                aria-invalid={signupErrorVisible}
                aria-describedby={signupErrorVisible
                  ? "signup-auth-error"
                  : undefined}
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
                aria-invalid={signupErrorVisible}
                aria-describedby={signupErrorVisible
                  ? "signup-auth-error"
                  : undefined}
              />
            </div>

            <div>
              <label
                for="signup-password"
                class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-2"
              >
                Password <span class="text-danger-600" aria-hidden="true"
                  >*</span
                >
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
                aria-invalid={signupErrorVisible}
                aria-describedby={signupErrorVisible
                  ? "signup-auth-error"
                  : undefined}
              />
            </div>

            {#if signupErrorVisible}
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
      {/if}
    </div>
  </main>
</div>
