import type { User, ArrService, ArrServiceInput, Movie, MovieSearchResult, MovieProviders, TvSeries, TvSeriesSearchResult, TvSeriesProviders, StreamingProvider } from "../types";

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

// Movies

export async function searchMovies(query: string): Promise<MovieSearchResult[]> {
  return request<MovieSearchResult[]>("GET", `/api/movies/search?q=${encodeURIComponent(query)}`);
}

export async function listMovies(): Promise<Movie[]> {
  return request<Movie[]>("GET", "/api/movies");
}

export async function likeMovie(movie: {
  tmdb_id: number;
  title: string;
  overview?: string;
  year?: number;
  poster_url?: string;
}): Promise<Movie> {
  return request<Movie>("POST", `/api/movies/${movie.tmdb_id}/like`, movie);
}

export async function unlikeMovie(tmdbId: number): Promise<void> {
  await request<void>("DELETE", `/api/movies/${tmdbId}/like`);
}

export async function getMovieProviders(tmdbId: number): Promise<MovieProviders> {
  return request<MovieProviders>("GET", `/api/movies/${tmdbId}/providers`);
}

// TV Series

export async function searchTvSeries(query: string): Promise<TvSeriesSearchResult[]> {
  return request<TvSeriesSearchResult[]>("GET", `/api/tv-series/search?q=${encodeURIComponent(query)}`);
}

export async function listTvSeries(): Promise<TvSeries[]> {
  return request<TvSeries[]>("GET", "/api/tv-series");
}

export async function likeTvSeries(series: {
  tmdb_id: number;
  title: string;
  overview?: string;
  year?: number;
  poster_url?: string;
}): Promise<TvSeries> {
  return request<TvSeries>("POST", `/api/tv-series/${series.tmdb_id}/like`, series);
}

export async function unlikeTvSeries(tmdbId: number): Promise<void> {
  await request<void>("DELETE", `/api/tv-series/${tmdbId}/like`);
}

export async function getTvSeriesProviders(tmdbId: number): Promise<TvSeriesProviders> {
  return request<TvSeriesProviders>("GET", `/api/tv-series/${tmdbId}/providers`);
}

// Config

export interface ConfigStatus {
  tmdb_configured: boolean;
  oidc_configured: boolean;
  is_admin: boolean;
}

export async function getConfigStatus(): Promise<ConfigStatus> {
  return request<ConfigStatus>("GET", "/api/config/status");
}

export async function setTmdbApiKey(apiKey: string): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", "/api/config/tmdb-api-key", { api_key: apiKey });
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

// Watch Providers

export async function listWatchProviders(): Promise<StreamingProvider[]> {
  return request<StreamingProvider[]>("GET", "/api/watch-providers");
}

export async function getUserWatchProviders(): Promise<StreamingProvider[]> {
  return request<StreamingProvider[]>("GET", "/api/user/watch-providers");
}

export async function setUserWatchProviders(providerIds: number[]): Promise<StreamingProvider[]> {
  return request<StreamingProvider[]>("PUT", "/api/user/watch-providers", { provider_ids: providerIds });
}
