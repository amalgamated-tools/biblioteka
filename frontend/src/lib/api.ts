import type {
  User,
  Library,
  LibraryInput,
  Author,
  AuthorInput,
  Series,
  SeriesInput,
  Book,
  BookInput,
  BookSeriesEntry,
  BookFile,
  BookFileInput,
  PaginatedBooks,
} from "../types";

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

// Config

export interface ConfigStatus {
  oidc_configured: boolean;
  smtp_configured: boolean;
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

export async function setOidcConfig(
  config: SetOIDCConfigInput,
): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", "/api/config/oidc", config);
}

export async function createOidcLinkNonce(): Promise<string> {
  const data = await request<{ nonce: string }>(
    "POST",
    "/api/auth/oidc/link-nonce",
  );
  return data.nonce;
}

// Admin - User Management

// SMTP Config

export interface SMTPConfig {
  host: string;
  port: string;
  username: string;
  password_set: boolean;
  from: string;
  tls: string;
  env_override: boolean;
}

export interface SetSMTPConfigInput {
  host: string;
  port: string;
  username: string;
  password: string;
  from: string;
  tls: string;
}

export async function getSmtpConfig(): Promise<SMTPConfig> {
  return request<SMTPConfig>("GET", "/api/config/smtp");
}

export async function setSmtpConfig(
  config: SetSMTPConfigInput,
): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", "/api/config/smtp", config);
}

export async function testSmtpConfig(): Promise<{ message: string }> {
  return request<{ message: string }>("POST", "/api/config/smtp/test");
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

export async function setUserAdmin(
  userId: string,
  isAdmin: boolean,
): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", `/api/admin/users/${userId}`, {
    is_admin: isAdmin,
  });
}

// Libraries

export async function listLibraries(): Promise<Library[]> {
  return request<Library[]>("GET", "/api/libraries");
}

export async function createLibrary(input: LibraryInput): Promise<Library> {
  return request<Library>("POST", "/api/libraries", input);
}

export async function updateLibrary(
  id: string,
  input: LibraryInput,
): Promise<Library> {
  return request<Library>("PUT", `/api/libraries/${id}`, input);
}

export async function deleteLibrary(id: string): Promise<void> {
  await request<void>("DELETE", `/api/libraries/${id}`);
}

export async function listLibraryBooks(
  libraryId: string,
  limit = 50,
  offset = 0,
): Promise<PaginatedBooks> {
  return request<PaginatedBooks>(
    "GET",
    `/api/libraries/${libraryId}/books?limit=${limit}&offset=${offset}`,
  );
}

// Authors

export async function listAuthors(): Promise<Author[]> {
  return request<Author[]>("GET", "/api/authors");
}

export async function getAuthor(id: string): Promise<Author> {
  return request<Author>("GET", `/api/authors/${id}`);
}

export async function createAuthor(input: AuthorInput): Promise<Author> {
  return request<Author>("POST", "/api/authors", input);
}

export async function updateAuthor(
  id: string,
  input: AuthorInput,
): Promise<Author> {
  return request<Author>("PUT", `/api/authors/${id}`, input);
}

export async function deleteAuthor(id: string): Promise<void> {
  await request<void>("DELETE", `/api/authors/${id}`);
}

// Series

export async function listSeries(): Promise<Series[]> {
  return request<Series[]>("GET", "/api/series");
}

export async function getSeries(id: string): Promise<Series> {
  return request<Series>("GET", `/api/series/${id}`);
}

export async function createSeries(input: SeriesInput): Promise<Series> {
  return request<Series>("POST", "/api/series", input);
}

export async function updateSeries(
  id: string,
  input: SeriesInput,
): Promise<Series> {
  return request<Series>("PUT", `/api/series/${id}`, input);
}

export async function deleteSeries(id: string): Promise<void> {
  await request<void>("DELETE", `/api/series/${id}`);
}

// Books

export async function listBooks(
  limit = 50,
  offset = 0,
): Promise<PaginatedBooks> {
  return request<PaginatedBooks>(
    "GET",
    `/api/books?limit=${limit}&offset=${offset}`,
  );
}

export async function getBook(id: string): Promise<Book> {
  return request<Book>("GET", `/api/books/${id}`);
}

export async function createBook(input: BookInput): Promise<Book> {
  return request<Book>("POST", "/api/books", input);
}

export async function updateBook(id: string, input: BookInput): Promise<Book> {
  return request<Book>("PUT", `/api/books/${id}`, input);
}

export async function deleteBook(id: string): Promise<void> {
  await request<void>("DELETE", `/api/books/${id}`);
}

// Book Authors

export async function getBookAuthors(bookId: string): Promise<Author[]> {
  return request<Author[]>("GET", `/api/books/${bookId}/authors`);
}

export async function setBookAuthors(
  bookId: string,
  authorIds: string[],
): Promise<Author[]> {
  return request<Author[]>("PUT", `/api/books/${bookId}/authors`, {
    author_ids: authorIds,
  });
}

// Book Series

export async function getBookSeries(
  bookId: string,
): Promise<BookSeriesEntry[]> {
  return request<BookSeriesEntry[]>("GET", `/api/books/${bookId}/series`);
}

export async function setBookSeries(
  bookId: string,
  entries: { series_id: string; position?: number }[],
): Promise<BookSeriesEntry[]> {
  return request<BookSeriesEntry[]>("PUT", `/api/books/${bookId}/series`, {
    entries,
  });
}

// Book Files

export async function listBookFiles(bookId: string): Promise<BookFile[]> {
  return request<BookFile[]>("GET", `/api/books/${bookId}/files`);
}

export async function createBookFile(
  bookId: string,
  input: BookFileInput,
): Promise<BookFile> {
  return request<BookFile>("POST", `/api/books/${bookId}/files`, input);
}

export async function getBookFile(id: string): Promise<BookFile> {
  return request<BookFile>("GET", `/api/book-files/${id}`);
}

export async function deleteBookFile(id: string): Promise<void> {
  await request<void>("DELETE", `/api/book-files/${id}`);
}

// API Keys

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  last_used_at: string | null;
  created_at: string;
}

export interface APIKeyCreateResponse extends APIKey {
  key: string;
}

export async function listAPIKeys(): Promise<APIKey[]> {
  return request<APIKey[]>("GET", "/api/api-keys");
}

export async function createAPIKey(
  name: string,
): Promise<APIKeyCreateResponse> {
  return request<APIKeyCreateResponse>("POST", "/api/api-keys", { name });
}

export async function deleteAPIKey(id: string): Promise<void> {
  await request<void>("DELETE", `/api/api-keys/${id}`);
}

// Kobo Sync Tokens

export interface KoboToken {
  id: string;
  user_id: string;
  name: string;
  token_hash: string;
  created_at: string;
}

export async function listKoboTokens(): Promise<KoboToken[]> {
  return request<KoboToken[]>("GET", "/api/kobo/tokens");
}

export interface KoboTokenCreateResponse extends KoboToken {
  token: string;
}

export async function createKoboToken(
  name: string,
): Promise<KoboTokenCreateResponse> {
  return request<KoboTokenCreateResponse>("POST", "/api/kobo/tokens", { name });
}

export async function deleteKoboToken(id: string): Promise<void> {
  await request<void>("DELETE", `/api/kobo/tokens/${id}`);
}

// Version

export async function getVersion(): Promise<string> {
  const data = await request<{ version: string }>("GET", "/api/version");
  return data.version;
}
