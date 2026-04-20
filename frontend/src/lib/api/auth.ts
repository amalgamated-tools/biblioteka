import type { AuthResponse, User } from "../../types";
import { clearToken, request, setToken } from "./core";

export async function signup(
  name: string,
  email: string,
  password: string,
): Promise<AuthResponse> {
  const data = await request<AuthResponse>("POST", "/api/auth/signup", {
    name,
    email,
    password,
  });
  setToken(data.token);
  return data;
}

export async function login(
  email: string,
  password: string,
): Promise<AuthResponse> {
  const data = await request<AuthResponse>("POST", "/api/auth/login", {
    email,
    password,
  });
  setToken(data.token);
  return data;
}

export async function getMe(): Promise<User> {
  return request<User>("GET", "/api/auth/me");
}

export async function logout(): Promise<void> {
  try {
    await request<{ message: string }>("POST", "/api/auth/logout");
  } catch {
    // Best-effort; cookie may already be cleared or expired.
  } finally {
    clearToken();
  }
}

async function getFeatureEnabled(
  path: string,
  signal?: AbortSignal,
): Promise<boolean> {
  const data = await request<{ enabled: boolean }>(
    "GET",
    path,
    undefined,
    signal,
  );
  return data.enabled === true;
}

export async function getOidcEnabled(signal?: AbortSignal): Promise<boolean> {
  return getFeatureEnabled("/api/auth/oidc/enabled", signal);
}

export async function getSignupEnabled(signal?: AbortSignal): Promise<boolean> {
  return getFeatureEnabled("/api/auth/signup/enabled", signal);
}

export async function createOidcLinkNonce(): Promise<string> {
  const data = await request<{ nonce: string }>(
    "POST",
    "/api/auth/oidc/link-nonce",
  );
  return data.nonce;
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  await request<{ message: string }>("PUT", "/api/auth/password", {
    currentPassword,
    newPassword,
  });
}

export async function updateProfile(name: string): Promise<User> {
  return request<User>("PUT", "/api/auth/me", { name });
}
