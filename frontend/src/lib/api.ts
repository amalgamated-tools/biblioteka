import type { User, ArrService, ArrServiceInput, Library, LibraryInput } from "../types";

const TOKEN_KEY = "biblioteka_token";

function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function hasToken(): boolean {
  return !!localStorage.getItem(TOKEN_KEY);
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const contentType = res.headers.get("content-type") || "";
  let data: unknown;

  if (contentType.includes("application/json")) {
    try {
      data = await res.json();
    } catch {
      const text = await res.text();
      data = text ? { error: text } : {};
    }
  } else {
    const text = await res.text();
    data = text ? { error: text } : {};
  }

  if (!res.ok) {
    const errorValue =
      typeof data === "object" && data !== null && "error" in data
        ? (data as { error?: unknown }).error
        : undefined;
    const message =
      typeof errorValue === "string" && errorValue.length > 0
        ? errorValue
        : res.statusText || "Request failed";
    throw new ApiError(message, res.status);
  }

  return data as T;
}

// Auth

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

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  await request<{ message: string }>("PUT", "/api/auth/password", {
    currentPassword,
    newPassword,
  });
}

// Arr Services

export async function listArrServices(): Promise<ArrService[]> {
  return request<ArrService[]>("GET", "/api/arr-services");
}

export async function createArrService(
  input: ArrServiceInput,
): Promise<ArrService> {
  return request<ArrService>("POST", "/api/arr-services", input);
}

export async function updateArrService(
  id: string,
  input: ArrServiceInput,
): Promise<ArrService> {
  return request<ArrService>("PUT", `/api/arr-services/${id}`, input);
}

export async function deleteArrService(id: string): Promise<void> {
  await request<void>("DELETE", `/api/arr-services/${id}`);
}

// Config

export interface ConfigStatus {
  oidc_configured: boolean;
  is_admin: boolean;
}

export async function getConfigStatus(): Promise<ConfigStatus> {
  return request<ConfigStatus>("GET", "/api/config/status");
}

// OIDC Config

export interface OIDCConfig {
  issuer_url: string;
  client_id: string;
  client_secret_set: boolean;
  redirect_uri: string;
}

export interface SetOIDCConfigInput {
  issuer_url: string;
  client_id: string;
  client_secret: string;
  redirect_uri: string;
}

export async function getOidcConfig(): Promise<OIDCConfig> {
  return request<OIDCConfig>("GET", "/api/config/oidc");
}

export async function setOidcConfig(config: SetOIDCConfigInput): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", "/api/config/oidc", config);
}

export async function createOidcLinkNonce(): Promise<string> {
  const data = await request<{ nonce: string }>("POST", "/api/auth/oidc/link-nonce");
  return data.nonce;
}

// Admin - User Management

export interface AdminUser {
  id: string;
  name: string;
  email: string;
  is_admin: boolean;
  oidc_linked: boolean;
  created_at: string;
}

export async function listUsers(): Promise<AdminUser[]> {
  return request<AdminUser[]>("GET", "/api/admin/users");
}

export async function setUserAdmin(userId: string, isAdmin: boolean): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", `/api/admin/users/${userId}`, { is_admin: isAdmin });
}

// Libraries

export async function listLibraries(): Promise<Library[]> {
  return request<Library[]>("GET", "/api/libraries");
}

export async function createLibrary(input: LibraryInput): Promise<Library> {
  return request<Library>("POST", "/api/libraries", input);
}

export async function updateLibrary(id: string, input: LibraryInput): Promise<Library> {
  return request<Library>("PUT", `/api/libraries/${id}`, input);
}

export async function deleteLibrary(id: string): Promise<void> {
  await request<void>("DELETE", `/api/libraries/${id}`);
}
