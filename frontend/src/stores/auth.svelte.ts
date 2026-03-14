import type { User } from "../types";
import * as api from "../lib/api";

class AuthStore {
  user: User | null = $state(null);
  loading = $state(true);
  oidcLinkError: string | null = $state(null);

  async init(): Promise<void> {
    // Check for OIDC callback token in URL
    const params = new URLSearchParams(window.location.search);
    const oidcToken = params.get("token");
    if (oidcToken) {
      api.setToken(oidcToken);
      window.history.replaceState({}, "", window.location.pathname);
    }

    // Check for OIDC link callback (success or error)
    const oidcLinked = params.get("oidc_linked");
    const linkError = params.get("oidc_link_error");
    if (oidcLinked || linkError) {
      if (linkError) {
        this.oidcLinkError = linkError;
      }
      window.history.replaceState({}, "", window.location.pathname);
    }

    if (api.hasToken()) {
      try {
        const u = await api.getMe();
        this.user = u;
      } catch {
        api.clearToken();
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
    api.clearToken();
    this.user = null;
  }
}

export const authStore = new AuthStore();
