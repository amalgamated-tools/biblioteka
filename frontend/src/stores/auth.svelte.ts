import type { User } from "../types";
import * as api from "../lib/api";

class AuthStore {
  user: User | null = $state(null);
  loading = $state(true);
  oidcLinkError: string | null = $state(null);

  async init(): Promise<void> {
    // Check for OIDC link callback (success or error)
    const params = new URLSearchParams(window.location.search);
    const oidcLinked = params.get("oidc_linked");
    const linkError = params.get("oidc_link_error");
    const oidcLogin = params.get("oidc_login");
    if (oidcLinked || linkError) {
      if (linkError) {
        this.oidcLinkError = linkError;
      }
      window.history.replaceState(
        {},
        "",
        window.location.pathname + "#settings",
      );
    } else if (oidcLogin) {
      // Clean up the OIDC login marker from the URL.
      window.history.replaceState({}, "", window.location.pathname);
    }

    // Always attempt to load the current user. Auth may come from a
    // localStorage token (normal login/signup) or an HttpOnly cookie (OIDC).
    try {
      this.user = await api.getMe();
    } catch {
      // If a localStorage token was sent and rejected, clear it and
      // retry — a valid HttpOnly cookie may still authenticate.
      if (api.hasToken()) {
        api.clearToken();
        try {
          this.user = await api.getMe();
        } catch {
          // Not authenticated via cookie either; stay logged out.
        }
      }
    }
    this.loading = false;
  }

  async signIn(
    email: string,
    password: string,
  ): Promise<{ error: Error | null }> {
    try {
      const result = await api.login(email, password);
      this.user = result.user;
      return { error: null };
    } catch (error) {
      return { error: error as Error };
    }
  }

  async signUp(
    name: string,
    email: string,
    password: string,
  ): Promise<{ error: Error | null }> {
    try {
      const result = await api.signup(name, email, password);
      this.user = result.user;
      return { error: null };
    } catch (error) {
      return { error: error as Error };
    }
  }

  async signOut(): Promise<void> {
    // Clear local auth state immediately so UI updates are instantaneous.
    this.user = null;
    api.clearToken();
    try {
      await api.logout();
    } catch {
      // Ignore logout failures; local sign-out has already completed.
    }
  }
}

export const authStore = new AuthStore();
