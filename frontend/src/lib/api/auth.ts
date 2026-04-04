import type { User } from "../../types";
import { request, setToken } from "./core";

interface AuthResponse {
  token: string;
  user: User;
}

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
  }
}

export async function getOidcEnabled(): Promise<boolean> {
  const data = await request<{ enabled: boolean }>(
    "GET",
    "/api/auth/oidc/enabled",
  );
  return data.enabled === true;
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
