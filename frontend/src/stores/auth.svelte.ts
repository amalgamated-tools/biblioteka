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
    if (oidcLinked || linkError) {
      if (linkError) {
        this.oidcLinkError = linkError;
      }
      window.history.replaceState(
        {},
        "",
        window.location.pathname + "#settings",
      );
    }

    // Try to load the current user. Auth may come from a localStorage token
    // (normal login/signup) or an HttpOnly cookie (OIDC callback).
    if (api.hasToken()) {
      try {
        const u = await api.getMe();
        this.user = u;
      } catch {
        // Token is present but invalid. Clear it and retry using only cookie-based auth.
        api.clearToken();
        try {
          const u = await api.getMe();
          this.user = u;
        } catch {
          // Not authenticated via cookie either; stay logged out.
        }
      }
    } else {
      // No localStorage token — try cookie-based auth (e.g. after OIDC redirect)
      try {
        const u = await api.getMe();
        this.user = u;
      } catch {
        // Not authenticated via cookie either; stay logged out.
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
