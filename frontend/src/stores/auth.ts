import { writable } from "svelte/store";
import type { User } from "../types";
import * as api from "../lib/api";

export const user = writable<User | null>(null);
export const authLoading = writable(true);
export const oidcLinkError = writable<string | null>(null);

export async function initAuth(): Promise<void> {
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
      oidcLinkError.set(linkError);
    }
    window.history.replaceState({}, "", window.location.pathname + "#settings");
  }

  if (api.hasToken()) {
    try {
      const u = await api.getMe();
      user.set(u);
    } catch {
      api.clearToken();
    }
  }
  authLoading.set(false);
}

export async function signIn(
  email: string,
  password: string,
): Promise<{ error: Error | null }> {
  try {
    const result = await api.login(email, password);
    user.set(result.user);
    return { error: null };
  } catch (error) {
    return { error: error as Error };
  }
}

export async function signUp(
  name: string,
  email: string,
  password: string,
): Promise<{ error: Error | null }> {
  try {
    const result = await api.signup(name, email, password);
    user.set(result.user);
    return { error: null };
  } catch (error) {
    return { error: error as Error };
  }
}

export async function signOut(): Promise<void> {
  api.clearToken();
  user.set(null);
}
